package main

import (
	//"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

type bot struct {
	api        *tba.Bot
	flashcards map[int64]map[string]map[string]string
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

func output(out chan msg, b *bot) {
	for {
		select {

		case m := <-out:
			SendMessage(b, m.chatID, m.text, defaultSendOpt())
		}
	}
}

// GetAnswer reads data from users message
func GetAnswer(b *bot, chatID int64, answer chan string) {

	b.api.Handle(tba.OnText, func(m *tba.Message) {

		if chatID == m.Chat.ID {
			answer <- m.Text
		}
	})
}

func SendMessage(b *bot, chat int64, message string, sendOpt *tba.SendOptions) {

	tmpChat := tba.Chat{ID: chat, Title: "", FirstName: "", LastName: "", Type: "", Username: ""}
	_, err := b.api.Send(&tmpChat, message, sendOpt)

	if err != nil {
		log.Printf("Could not send message %s to %d", message, chat)
	}
}

//TODO: make it work for multiple users at the same time, then make edit and delete
func addFlashcard(flashcards map[int64]map[string]map[string]string, chatID int64, out chan msg, in chan string) {

	var subject string
	var term string
	var definition string
	out <- msg{chatID, "Podaj temat fiszki"}
	select {
	case a := <-in:
		subject = a
	}

	out <- msg{chatID, "Podaj pojecie"}
	select {
	case a := <-in:
		term = a
	}

	if _, ok := flashcards[chatID][subject][term]; ok {
		out <- msg{chatID, "Fiszka juz istnieje"}
		return
	}

	out <- msg{chatID, "Podaj definicje"}
	select {
	case a := <-in:
		definition = a
	}

	if flashcards[chatID] == nil {

		flashcards[chatID] = make(map[string]map[string]string)
	}

	if flashcards[chatID][subject] == nil {

		flashcards[chatID][subject] = make(map[string]string)
	}

	flashcards[chatID][subject][term] = definition

	flashcardsJson, err := json.Marshal(flashcards)

	if err != nil {
		log.Print("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, flashcardsJson, 0644)

	if err != nil {
		log.Print("Could not write flashcards")
		out <- msg{chatID, "Wystapil problem, sprobuj pozniej"}
		return
	}

	out <- msg{chatID, "Dodano fiszke"}
}

func findFlashcard(flashcards map[string]map[string]string, m *tba.Message, output chan msg) {

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

func (b *bot) Run() {

	channels := make(map[int64]chan string)
	outChannel := make(chan msg)

	go output(outChannel, b)

	b.api.Handle("/version", func(m *tba.Message) {

		outChannel <- msg{m.Chat.ID, "version 0.0.3"}
	})

	//single line commands don't stop routines
	b.api.Handle("/fiszka", func(m *tba.Message) {

		findFlashcard(b.flashcards[m.Chat.ID], m, outChannel)
	})

	b.api.Handle("/dodajfiszke", func(m *tba.Message) {

		channels[m.Chat.ID] = make(chan string)
		go addFlashcard(b.flashcards, m.Chat.ID, outChannel, channels[m.Chat.ID])
	})

	b.api.Handle(tba.OnText, func(m *tba.Message) {

		channels[m.Chat.ID] <- m.Text
	})

	b.api.Start()
}

func NewBot(token string) *bot {

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

	log.Printf("Bot authorized")
	return &bot{tb, flashcards}
}
