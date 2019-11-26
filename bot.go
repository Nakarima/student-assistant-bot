package main

import (
	//"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

//Bot is main structure with access to api, users data and necessary channels
type Bot struct {
	api           *tba.Bot
	flashcards    map[int64]map[string]map[string]string
	input         map[int64]chan string
	inactiveInput chan int64
	output        chan msg
}

type msg struct {
	chatID int64
	text   string
}

const flashcardsFileName = "flashcards.json"

func defaultSendOpt() *tba.SendOptions {

	return &tba.SendOptions{}
}

func ensureDataFileExists(fileName string) {

	if _, err := os.Stat(fileName); err != nil {

		err = ioutil.WriteFile(fileName, []byte("{}"), 0644)
		if err != nil {

			log.Fatalf("could not create %s", fileName)
		}
	}
}

func output(b *Bot) {

	for {

		select {

		case m := <-b.output:
			sendMessage(b, m.chatID, m.text, defaultSendOpt())
		}
	}
}

func inputKiller(c chan int64, m map[int64]chan string) {
	for {

		select {

		case id := <-c:
			delete(m, id)
		}
	}
}
func sendMessage(b *Bot, chat int64, message string, sendOpt *tba.SendOptions) {

	tmpChat := tba.Chat{ID: chat, Title: "", FirstName: "", LastName: "", Type: "", Username: ""}
	_, err := b.api.Send(&tmpChat, message, sendOpt)

	if err != nil {

		log.Printf("Could not send message %s to %d", message, chat)
	}
}

func getAnswer(in chan string) (string, error) {

	timeout := 0
	for timeout < 1200 {

		select {

		case a := <-in:
			return a, nil
		default:
			time.Sleep(250 * time.Millisecond)
			timeout++
		}
	}

	return "", errors.New("timeout")
}

func dialog(out chan msg, chatID int64, question string, in chan string) (string, error) {

	out <- msg{chatID, question}
	a, err := getAnswer(in)
	if err != nil {

		out <- msg{chatID, "Przekroczono czas odpowiedzi"}
	}
	return a, err

}

func addFlashcard(flashcards map[int64]map[string]map[string]string, chatID int64, out chan msg, in chan string, state chan int64) {

	topic, err := dialog(out, chatID, "Podaj temat", in)
	if err != nil {
		state <- chatID
		return
	}

	term, err := dialog(out, chatID, "Podaj pojecie", in)
	if err != nil {
		state <- chatID
		return
	}

	if _, ok := flashcards[chatID][topic][term]; ok {

		out <- msg{chatID, "Fiszka juz istnieje"}
		state <- chatID
		return
	}

	definition, err := dialog(out, chatID, "Podaj definicje", in)
	if err != nil {
		state <- chatID
		return
	}

	if flashcards[chatID] == nil {

		flashcards[chatID] = make(map[string]map[string]string)
	}

	if flashcards[chatID][topic] == nil {

		flashcards[chatID][topic] = make(map[string]string)
	}

	flashcards[chatID][topic][term] = definition

	flashcardsJSON, err := json.Marshal(flashcards)
	if err != nil {
		log.Print("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, flashcardsJSON, 0644)
	if err != nil {
		log.Print("Could not write flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Dodano fiszke"}
	state <- chatID
}

func displayFlashcard(flashcards map[string]map[string]string, m *tba.Message, output chan msg) {

	chatID := m.Chat.ID

	if m.Text == "/fiszka" {
		output <- msg{chatID, "Podaj pojecie po spacji"}
		return
	}

	term := strings.ReplaceAll(m.Text, "/fiszka ", "")
	flashcardFound := false

	for subject, val := range flashcards {

		if definition, ok := val[term]; ok {

			flashcardFound = true
			output <- msg{
				chatID,
				subject + ", " + term + " - " + definition,
			}
		}
	}

	if !flashcardFound {

		output <- msg{chatID, "Nie znaleziono pojecia"}
	}
}

func deleteFlashcard(flashcards map[int64]map[string]map[string]string, chatID int64, out chan msg, in chan string, state chan int64) {

	topic, err := dialog(out, chatID, "Podaj temat", in)
	if err != nil {
		state <- chatID
		return
	}

	term, err := dialog(out, chatID, "Podaj pojecie", in)
	if err != nil {
		state <- chatID
		return
	}

	if _, ok := flashcards[chatID][topic][term]; !ok {

		out <- msg{chatID, "Fiszka nie istnieje"}
		state <- chatID
		return
	}

	delete(flashcards[chatID][topic], term)

	if flashcards[chatID][topic] == nil {
		delete(flashcards[chatID], topic)
	}

	flashcardsJSON, err := json.Marshal(flashcards)
	if err != nil {
		log.Print("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, flashcardsJSON, 0644)
	if err != nil {
		log.Print("Could not write flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Usunieto fiszke"}
	state <- chatID
}

//Run starts all handlers and listeners for bot
func (b *Bot) Run() {

	go output(b)

	go inputKiller(b.inactiveInput, b.input)

	b.api.Handle("/version", func(m *tba.Message) {

		b.output <- msg{m.Chat.ID, "version 0.0.5"}
	})

	//single line commands don't stop routines
	b.api.Handle("/fiszka", func(m *tba.Message) {

		displayFlashcard(b.flashcards[m.Chat.ID], m, b.output)
	})

	b.api.Handle("/dodajfiszke", func(m *tba.Message) {

		b.input[m.Chat.ID] = make(chan string)
		go addFlashcard(b.flashcards, m.Chat.ID, b.output, b.input[m.Chat.ID], b.inactiveInput)
	})

	b.api.Handle("/usunfiszke", func(m *tba.Message) {

		b.input[m.Chat.ID] = make(chan string)
		go deleteFlashcard(b.flashcards, m.Chat.ID, b.output, b.input[m.Chat.ID], b.inactiveInput)
	})

	b.api.Handle(tba.OnText, func(m *tba.Message) {

		if d, ok := b.input[m.Chat.ID]; ok {

			d <- m.Text
		}
	})

	b.api.Start()
}

//NewBot creates new bot instance under given telegram api token
func NewBot(token string) *Bot {

	tb, err := tba.NewBot(tba.Settings{

		Token:  token,
		Poller: &tba.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {

		log.Fatal("Could not create bot")
	}

	flashcards := make(map[int64]map[string]map[string]string)
	ensureDataFileExists(flashcardsFileName)
	flashcardsData, err := ioutil.ReadFile(flashcardsFileName)

	if err != nil {

		log.Fatalf("Could not read %s", flashcardsFileName)
	}

	err = json.Unmarshal([]byte(flashcardsData), &flashcards)

	if err != nil {

		log.Fatalf("Could not decode %s", flashcardsFileName)
	}

	input := make(map[int64]chan string)
	inactiveInput := make(chan int64)
	output := make(chan msg)

	log.Printf("Bot authorized")
	return &Bot{tb, flashcards, input, inactiveInput, output}
}
