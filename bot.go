package main

import (
	"log"
	"time"

	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

type bot struct {
	api *tba.Bot
}

func (b *bot) Run() {
	b.api.Handle("/version", func(m *tba.Message) {
		_, err := b.api.Send(m.Chat, "v 0.0.1")
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
