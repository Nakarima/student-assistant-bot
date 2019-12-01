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
type chatid int64
type topic string
type flashcard map[string]string
type flashcards map[chatid]map[topic]flashcard

//Bot struct stores api, data and all necessary channels
type Bot struct {
	api           *tba.Bot
	flashcards    flashcards
	input         map[chatid]chan string
	inactiveInput chan chatid
	output        chan msg
}

type msg struct {
	chatID chatid
	text   string
}

const flashcardsFileName = "flashcards.json"

func defaultSendOpt() *tba.SendOptions {

	return &tba.SendOptions{}

}

//TODO: logrus with special logging for some minor errors like here
func ensureDataFileExists(fileName string) error {

	if _, err := os.Stat(fileName); err != nil {
		err = ioutil.WriteFile(fileName, []byte("{}"), 0644)
		if err != nil {
			log.Fatalf("could not create %s", fileName)
			return err
		}
	}
	return nil

}

func output(b *Bot) {

	for m := range b.output {
		_ = sendMessage(b, m.chatID, m.text, defaultSendOpt())
	}

}

func inputKiller(c chan chatid, m map[chatid]chan string) {

	for id := range c {
		delete(m, id)
	}

}

func sendMessage(b *Bot, chat chatid, message string, sendOpt *tba.SendOptions) error {

	tmpChat := tba.Chat{ID: int64(chat), Title: "", FirstName: "", LastName: "", Type: "", Username: ""}
	_, err := b.api.Send(&tmpChat, message, sendOpt)

	if err != nil {
		log.Printf("Could not send message %s to %d", message, chat)
	}

	return err

}

func getAnswer(in chan string) (string, error) {

	select {
	case a := <-in:
		return a, nil
	case <-time.After(10 * time.Minute):
		return "", errors.New("timeout")
	}

}

func dialog(out chan msg, chatID chatid, question string, in chan string) (string, error) {

	out <- msg{chatID, question}
	a, err := getAnswer(in)

	if err != nil {
		out <- msg{chatID, "Przekroczono czas odpowiedzi"}
	}

	return a, err

}

func addFlashcard(fc flashcards, chatID chatid, out chan msg, in chan string, state chan chatid) {

	t, err := dialog(out, chatID, "Podaj temat", in)
	if err != nil {
		state <- chatID
		return
	}
	top := topic(t)

	term, err := dialog(out, chatID, "Podaj pojecie", in)
	if err != nil {
		state <- chatID
		return
	}

	if _, ok := fc[chatID][topic(top)][term]; ok {
		out <- msg{chatID, "Fiszka juz istnieje, edytuj za pomoca /edytujfiszke"}
		state <- chatID
		return
	}

	definition, err := dialog(out, chatID, "Podaj definicje", in)
	if err != nil {
		state <- chatID
		return
	}

	if fc[chatID] == nil {
		fc[chatID] = make(map[topic]flashcard)
	}

	if fc[chatID][top] == nil {
		fc[chatID][top] = make(flashcard)
	}

	fc[chatID][top][term] = definition

	fcJSON, err := json.Marshal(fc)
	if err != nil {
		log.Print("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, fcJSON, 0644)
	if err != nil {
		log.Print("Could not write flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Dodano fiszke"}
	state <- chatID

}

func displayFlashcard(fc flashcards, m *tba.Message, output chan msg) {

	chatID := chatid(m.Chat.ID)

	if m.Text == "/fiszka" {
		output <- msg{chatID, "Podaj pojecie po spacji"}
		return
	}

	term := strings.ReplaceAll(m.Text, "/fiszka ", "")
	flashcardFound := false

	for top, val := range fc[chatID] {
		if definition, ok := val[term]; ok {
			flashcardFound = true
			output <- msg{
				chatID,
				string(top) + ", " + term + " - " + definition,
			}
		}
	}

	if !flashcardFound {
		output <- msg{chatID, "Nie znaleziono pojecia"}
	}

}

func deleteFlashcard(fc flashcards, chatID chatid, out chan msg, in chan string, state chan chatid) {

	t, err := dialog(out, chatID, "Podaj temat", in)
	if err != nil {
		state <- chatID
		return
	}
	top := topic(t)

	term, err := dialog(out, chatID, "Podaj pojecie", in)
	if err != nil {
		state <- chatID
		return
	}

	if _, ok := fc[chatID][top][term]; !ok {

		out <- msg{chatID, "Fiszka nie istnieje"}
		state <- chatID
		return
	}

	delete(fc[chatID][top], term)

	if fc[chatID][top] == nil {
		delete(fc[chatID], top)
	}

	fcJSON, err := json.Marshal(fc)
	if err != nil {
		log.Print("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, fcJSON, 0644)
	if err != nil {
		log.Print("Could not write flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Usunieto fiszke"}
	state <- chatID

}
func editFlashcard(fc flashcards, chatID chatid, out chan msg, in chan string, state chan chatid) {

	t, err := dialog(out, chatID, "Podaj temat", in)
	if err != nil {
		state <- chatID
		return
	}
	top := topic(t)

	term, err := dialog(out, chatID, "Podaj pojecie", in)
	if err != nil {
		state <- chatID
		return
	}

	if _, ok := fc[chatID][top][term]; !ok {
		out <- msg{chatID, "Fiszka nie istnieje"}
		state <- chatID
		return
	}

	definition, err := dialog(out, chatID, "Podaj definicje", in)
	if err != nil {
		state <- chatID
		return
	}

	fc[chatID][top][term] = definition

	fcJSON, err := json.Marshal(fc)
	if err != nil {
		log.Print("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, fcJSON, 0644)
	if err != nil {
		log.Print("Could not write flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Edytowano fiszke"}
	state <- chatID

}

//Run starts all handlers and listeners for bot
func (b *Bot) Run() {

	go output(b)

	go inputKiller(b.inactiveInput, b.input)

	b.api.Handle("/version", func(m *tba.Message) {
		b.output <- msg{chatid(m.Chat.ID), "version 0.0.6"}
	})

	//single line commands don't stop routines
	b.api.Handle("/fiszka", func(m *tba.Message) {
		displayFlashcard(b.flashcards, m, b.output)
	})

	b.api.Handle("/dodajfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		b.input[chatID] = make(chan string)
		go addFlashcard(b.flashcards, chatID, b.output, b.input[chatID], b.inactiveInput)
	})

	b.api.Handle("/usunfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		b.input[chatID] = make(chan string)
		go deleteFlashcard(b.flashcards, chatID, b.output, b.input[chatID], b.inactiveInput)
	})

	b.api.Handle("/edytujfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		b.input[chatID] = make(chan string)
		go editFlashcard(b.flashcards, chatID, b.output, b.input[chatID], b.inactiveInput)
	})

	b.api.Handle(tba.OnText, func(m *tba.Message) {
		if d, ok := b.input[chatid(m.Chat.ID)]; ok {
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

	flashcards := make(flashcards)
	_ = ensureDataFileExists(flashcardsFileName)
	flashcardsData, err := ioutil.ReadFile(flashcardsFileName)

	if err != nil {
		log.Fatalf("Could not read %s", flashcardsFileName)
	}

	err = json.Unmarshal([]byte(flashcardsData), &flashcards)

	if err != nil {
		log.Fatalf("Could not decode %s", flashcardsFileName)
	}

	input := make(map[chatid]chan string)
	inactiveInput := make(chan chatid)
	output := make(chan msg)

	log.Printf("Bot authorized")
	return &Bot{tb, flashcards, input, inactiveInput, output}

}
