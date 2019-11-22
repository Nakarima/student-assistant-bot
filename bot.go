package main

import (
	//"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"time"

	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

type bot struct {
	api        *tba.Bot
	flashcards map[string]string
}

const flashcardsFileName = "flashcards.json"

func defaultSendOpt(m *tba.Message) *tba.SendOptions {
	return &tba.SendOptions{
		ReplyTo: m,
	}
}

func SendMessage(b *bot, chat int64, message string, sendOpt *tba.SendOptions) error {
	tmpChat := tba.Chat{ID: chat, Title: "", FirstName: "", LastName: "", Type: "", Username: ""}
	_, err := b.api.Send(&tmpChat, message, sendOpt)

	return err
}

func (b *bot) Run() {
	b.api.Handle("/version", func(m *tba.Message) {
		err := SendMessage(b, m.Chat.ID, "version 0.0.1", defaultSendOpt(m))
		if err != nil {
			log.Printf("error sending version to %d", m.Chat.ID)
		}
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

	flashcards := make(map[string]string)
	flashcardsData, err := ioutil.ReadFile(flashcardsFileName)
	if err != nil {
		log.Printf("Could not read %s", flashcardsFileName)
	} else {
		err = json.Unmarshal([]byte(flashcardsData), &flashcards)
		if err != nil {
			log.Printf("Could not decode %s", flashcardsFileName)
		}
	}

	log.Printf("Bot authorized")
	return &bot{tb, flashcards}
}
