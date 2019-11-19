package main

import (
	"log"
	"time"

	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

type bot struct {
	api *tba.Bot
}

func DefaultSendOpt(m *tba.Message) *tba.SendOptions {
	return &tba.SendOptions{
		ReplyTo: m,
	}
}

func SendMessage(b *bot, chat int64, message string, sendOpt *tba.SendOptions) error {
	tmpChat := tba.Chat{ID: chat, Title: "", FirstName: "", LastName: "", Type: "", Username: ""}
	_, err := b.api.Send(&tmpChat, message, sendOpt)
	if err != nil {
		print("Error sending message to %d", chat)
	}
	return err
}

func (b *bot) Run() {
	b.api.Handle("/version", func(m *tba.Message) {
		err := SendMessage(b, m.Chat.ID, "version 0.0.1", DefaultSendOpt(m))
		if err != nil {
			log.Fatal("error sending version")
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

	log.Printf("Bot authorized")
	return &bot{tb}
}
