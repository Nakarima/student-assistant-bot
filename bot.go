package main

import (
	//"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"time"

	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

type bot struct {
	api        *tba.Bot
	flashcards map[int64]map[string]map[string]string
}

const flashcardsFileName = "flashcards.json"

func defaultSendOpt(m *tba.Message) *tba.SendOptions {
	return &tba.SendOptions{
		ReplyTo: m,
	}
}

func ensureDataFileExists(fileName string) {
	if _, err := os.Stat(fileName); err != nil {
		err = ioutil.WriteFile(fileName, []byte("{}"), 0644)
		if err != nil {
			log.Fatalf("could not create %s", fileName)
		}
	}
}

func GetAnswer(b *bot, chatID int64, answer chan string) {
	b.api.Handle(tba.OnText, func(m *tba.Message) {
		if chatID == m.Chat.ID {
			answer <- m.Text
		}
	})
}

func SendMessage(b *bot, chat int64, message string, sendOpt *tba.SendOptions) error {
	tmpChat := tba.Chat{ID: chat, Title: "", FirstName: "", LastName: "", Type: "", Username: ""}
	_, err := b.api.Send(&tmpChat, message, sendOpt)

	return err
}

func addFlashcard(b *bot, chatID int64) {
	subjectChan := make(chan string)
	SendMessage(b, chatID, "Podaj temat fiszki", &tba.SendOptions{})
	GetAnswer(b, chatID, subjectChan)
	subject := <-subjectChan

	termChan := make(chan string)
	SendMessage(b, chatID, "Podaj pojecie", &tba.SendOptions{})
	GetAnswer(b, chatID, termChan)
	term := <-termChan

	definitionChan := make(chan string)
	SendMessage(b, chatID, "Podaj definicje", &tba.SendOptions{})
	GetAnswer(b, chatID, definitionChan)
	definition := <-definitionChan

	if b.flashcards[chatID] == nil {
		b.flashcards[chatID] = make(map[string]map[string]string)
	}
	if b.flashcards[chatID][subject] == nil {
		b.flashcards[chatID][subject] = make(map[string]string)
	}
	b.flashcards[chatID][subject][term] = definition

	log.Print(b.flashcards[chatID])

}

func (b *bot) Run() {
	b.api.Handle("/version", func(m *tba.Message) {
		err := SendMessage(b, m.Chat.ID, "version 0.0.1", defaultSendOpt(m))
		if err != nil {
			log.Printf("error sending version to %d", m.Chat.ID)
		}
	})
	b.api.Handle("/dodajfiszke", func(m *tba.Message) {
		go addFlashcard(b, m.Chat.ID)
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
