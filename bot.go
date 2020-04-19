package main

import (
	//"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

const funcs = `
/version - podaje aktualną wersje
/fiszka {nazwa} - podaje fiszke pod podaną nazwą
/dodajfiszke - uruchamia dialog dodawania fiszki
/usunfiszke - uruchamia dialog usuwania fiszki
/edytujfiszke - uruchamia dialog edytowania fiszki
/test -  uruchamia test wiedzy
/dodajprzypomnienie - uruchamia dialog dodawania przypomnienia
/pokazprzypomnienia - wypisuje listę aktualnych przypomnień
`

type chatid int64

//Bot struct stores api, data and all necessary channels
type Bot struct {
	api            *tba.Bot
	flashcardsData flashcardsData
	remindersData  remindersData
	input          map[chatid]chan string
	inactiveInput  chan chatid
	output         chan msg
}

type msg struct {
	chatID chatid
	text   string
}

func generateDialogLogger(chatID chatid) *log.Entry {
	return log.WithFields(log.Fields{
		"chat": chatID,
	})
}

func generateIoLogger(filename string, funcname string) *log.Entry {

	return log.WithFields(log.Fields{
		"file": filename,
		"func": funcname,
	})
}

func defaultSendOpt() *tba.SendOptions {

	return &tba.SendOptions{}

}

func ensureDataFileExists(fileName string) error {

	if _, err := os.Stat(fileName); err != nil {
		err = ioutil.WriteFile(fileName, []byte("{}"), 0644)
		if err != nil {
			log.WithFields(log.Fields{
				"file": fileName,
			}).Fatal("could not create file")
			return err
		}
	}
	return nil

}

func (b *Bot) handleOutput() {

	for m := range b.output {
		_ = b.sendMessage(m.chatID, m.text, defaultSendOpt())
	}

}

func (b *Bot) inputKiller() {

	for id := range b.inactiveInput {
		close(b.input[id])
		delete(b.input, id)
	}

}

func (b *Bot) sendMessage(chat chatid, message string, sendOpt *tba.SendOptions) error {

	tmpChat := tba.Chat{ID: int64(chat), Title: "", FirstName: "", LastName: "", Type: "", Username: ""}
	_, err := b.api.Send(&tmpChat, message, sendOpt)

	if err != nil {
		log.WithFields(log.Fields{
			"chat":    chat,
			"message": message,
		}).Error("Could not send message")
	}

	return err

}

func getAnswer(in chan string) (string, error) {

	select {
	case a := <-in:
		if a == "" {
			return a, errors.New("ended dialog")
		}
		return a, nil
	case <-time.After(10 * time.Minute):
		return "", errors.New("timeout")
	}

}

func (b *Bot) dialog(chatID chatid, question string) (string, error) {

	b.output <- msg{chatID, question}
	a, err := getAnswer(b.input[chatID])

	if err != nil {
		if err.Error() == "ended dialog" {
			return a, err
		}
		log.WithFields(log.Fields{
			"chat": chatID,
		}).Info("User did not answer in given time")
		b.output <- msg{chatID, "Przekroczono czas odpowiedzi"}
	}

	return a, err
}

func (b *Bot) help(chatID chatid) {
	b.output <- msg{chatID, funcs}
}

//Run starts all handlers and listeners for bot
func (b *Bot) Run() {

	go b.handleOutput()
	go b.inputKiller()
	b.setReminders()

	b.api.Handle("/version", func(m *tba.Message) {
		b.output <- msg{chatid(m.Chat.ID), "version 0.3.1"}
	})

	b.api.Handle("/help", func(m *tba.Message) {
		go b.help(chatid(m.Chat.ID))
	})

	//single line commands don't stop routines
	b.api.Handle("/fiszka", func(m *tba.Message) {
		b.displayFlashcard(m)
	})

	b.api.Handle("/dodajfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.input[chatID]; ok {
			b.input[chatID] <- ""
			//TODO: make it without sleep
			time.Sleep(2 * time.Second)
		}
		b.input[chatID] = make(chan string)
		go b.addFlashcard(chatID)
	})

	b.api.Handle("/usunfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.input[chatID]; ok {
			b.input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.input[chatID] = make(chan string)
		go b.deleteFlashcard(chatID)
	})

	b.api.Handle("/edytujfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.input[chatID]; ok {
			b.input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.input[chatID] = make(chan string)
		go b.editFlashcard(chatID)
	})

	b.api.Handle("/test", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.input[chatID]; ok {
			b.input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.input[chatID] = make(chan string)
		go b.knowledgeTest(chatID)
	})

	b.api.Handle("/dodajprzypomnienie", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.input[chatID]; ok {
			b.input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.input[chatID] = make(chan string)
		go b.addReminder(chatID)
	})

	b.api.Handle("/pokazprzypomnienia", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)

		go b.showReminders(chatID)
	})

	b.api.Handle(tba.OnText, func(m *tba.Message) {
		if d, ok := b.input[chatid(m.Chat.ID)]; ok {
			d <- m.Text
		}
	})

	b.api.Start()

}

//NewBot creates new bot instance under given telegram api token
func NewBot(token string, env string) *Bot {

	if env == "prod" {
		log.SetFormatter(&log.JSONFormatter{})
		file, err := os.OpenFile("logrus.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Info("Failed to log to file, using stderr")
		} else {
			log.SetOutput(file)
		}
	}

	tb, err := tba.NewBot(tba.Settings{
		Token:  token,
		Poller: &tba.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal("Could not create bot")
	}

	flashcards := make(flashcardsData)
	_ = ensureDataFileExists(flashcardsFileName)
	flashcardsData, err := ioutil.ReadFile(flashcardsFileName)

	if err != nil {
		log.WithFields(log.Fields{
			"file": flashcardsFileName,
		}).Fatal("Could not read file")
	}

	err = json.Unmarshal([]byte(flashcardsData), &flashcards)

	if err != nil {
		log.WithFields(log.Fields{
			"file": flashcardsFileName,
		}).Fatal("Could not decode file")
	}

	reminders := make(remindersData)
	_ = ensureDataFileExists(remindersFileName)
	remindersData, err := ioutil.ReadFile(remindersFileName)

	if err != nil {
		log.WithFields(log.Fields{
			"file": remindersFileName,
		}).Fatal("Could not read file")
	}

	err = json.Unmarshal([]byte(remindersData), &reminders)

	if err != nil {
		log.WithFields(log.Fields{
			"file": remindersFileName,
		}).Fatal("Could not decode file")
	}

	input := make(map[chatid]chan string)
	inactiveInput := make(chan chatid)
	output := make(chan msg)

	log.Info("Bot authorized")
	return &Bot{tb, flashcards, reminders, input, inactiveInput, output}

}
