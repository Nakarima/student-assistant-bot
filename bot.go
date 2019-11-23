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

func SendMessage(b *bot, chat int64, message string, sendOpt *tba.SendOptions) {

	tmpChat := tba.Chat{ID: chat, Title: "", FirstName: "", LastName: "", Type: "", Username: ""}
	_, err := b.api.Send(&tmpChat, message, sendOpt)

	if err != nil {
		log.Printf("Could not send message %s to %d", message, chat)
	}
}

//TODO: make it work for multiple users at the same time, then make edit and delete
func addFlashcard(b *bot, chatID int64) {

	subjectChan := make(chan string)
	SendMessage(b, chatID, "Podaj temat fiszki", &tba.SendOptions{})
	GetAnswer(b, chatID, subjectChan)
	subject := <-subjectChan

	termChan := make(chan string)
	SendMessage(b, chatID, "Podaj pojecie", &tba.SendOptions{})
	GetAnswer(b, chatID, termChan)
	term := <-termChan

	if _, ok := b.flashcards[chatID][subject][term]; ok {
		SendMessage(b, chatID, "Fiszka juz istnieje", &tba.SendOptions{})
		return
	}

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

	flashcardsJson, err := json.Marshal(b.flashcards)

	if err != nil {
		log.Print("Could not encode flashcards")
		SendMessage(b, chatID, "Wystapil problem, sprobuj pozniej", &tba.SendOptions{})
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, flashcardsJson, 0644)

	if err != nil {
		log.Print("Could not write flashcards")
		SendMessage(b, chatID, "Wystapil problem, sprobuj pozniej", &tba.SendOptions{})
		return
	}

	SendMessage(b, chatID, "Dodano fiszke", &tba.SendOptions{})
}

func findFlashcard(b *bot, m *tba.Message) {

	chatID := m.Chat.ID

	if m.Text == "/fiszka" {
		SendMessage(b, chatID, "Podaj pojecie po spacji", defaultSendOpt(m))
		return
	}

	term := strings.ReplaceAll(m.Text, "/fiszka ", "")
	tmp := b.flashcards[chatID]
	flashcardFound := false

	for subject, val := range tmp {

		if definition, ok := val[term]; ok {

			flashcardFound = true
			SendMessage(
				b,
				chatID,
				subject+", "+term+" - "+definition,
				defaultSendOpt(m),
			)
		}
	}

	if !flashcardFound {

		SendMessage(b, chatID, "Nie znaleziono pojecia", defaultSendOpt(m))
	}
}

func (b *bot) Run() {

	b.api.Handle("/version", func(m *tba.Message) {

		SendMessage(b, m.Chat.ID, "version 0.0.3", defaultSendOpt(m))
	})

	b.api.Handle("/dodajfiszke", func(m *tba.Message) {

		go addFlashcard(b, m.Chat.ID)
	})

	b.api.Handle("/fiszka", func(m *tba.Message) {

		findFlashcard(b, m)
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
