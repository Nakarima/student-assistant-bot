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
/dodajzajecia - uruchamia dialog dodawania zajęć
/edytujzajecia - uruchamia dialog edytowania zajęć
/usunzajecia - uruchamia dialog usuwania zajęć
/plan - wypisuje plan zajec
/usunplan - uruchamia dialog usuwania zajęć
`

type chatid int64

// Bot struct stores api, data and all necessary channels.
// FlashcardsData stores all flashcards by chat ID.
// RemindersData stores all reminders by chat ID.
// SchedulesData stores all schedules by chat ID.
// Input is a channel for managing all messages from chats.
// InactiveInput is a channel that informs bot about exiting dialogs so he can make it available for other dialog.
// Output is a channel for sending message to chats.
type Bot struct {
	api            *tba.Bot
	FlashcardsData flashcardsData
	RemindersData  remindersData
	SchedulesData  schedulesData
	Input          map[chatid]chan string
	InactiveInput  chan chatid
	Output         chan Msg
}

// Msg is basic message struct. It stores desired chat ID and text message.
type Msg struct {
	chatID chatid
	text   string
}

// generateDialogLogger creates logger for dialog errors
func generateDialogLogger(chatID chatid) *log.Entry {
	return log.WithFields(log.Fields{
		"chat": chatID,
	})
}

// generateIoLogger creates logger for any file related errors.
func generateIoLogger(filename string, funcname string) *log.Entry {
	return log.WithFields(log.Fields{
		"file": filename,
		"func": funcname,
	})
}

// defaultSendOpt stores default config for sending messages to chat.
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

// HandleOutput listens for messages on Output channel and sends them to desired chat.
func (b *Bot) HandleOutput() {
	for m := range b.Output {
		_ = b.SendMessage(m.chatID, m.text, defaultSendOpt())
	}
}

// InputKiller listens for chat IDs on InactiveInput channel and then deletes desired Input channel.
func (b *Bot) InputKiller() {
	for id := range b.InactiveInput {
		close(b.Input[id])
		delete(b.Input, id)
	}
}

// SendMessage creates messages specific for telegram api and then sends them to desired chat. It wraps chatid to chat object, because it is requirment for tucnak's package.
func (b *Bot) SendMessage(chat chatid, message string, sendOpt *tba.SendOptions) error {
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

// getAnswer listens for users message and returns it. If user doesn't respond in given time it returns error. If input is empty string it will return error. It's good for ending opened dialog when we want to start a new one.
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

// Dialog handles basic user-bot interaction. Bot will ask given question, and then listen for user's answer. If everything is correct it will return answer.
func (b *Bot) Dialog(chatID chatid, question string) (string, error) {
	b.Output <- Msg{chatID, question}
	a, err := getAnswer(b.Input[chatID])

	if err != nil {
		if err.Error() == "ended dialog" {
			return a, err
		}
		log.WithFields(log.Fields{
			"chat": chatID,
		}).Info("User did not answer in given time")
		b.Output <- Msg{chatID, "Przekroczono czas odpowiedzi"}
	}

	return a, err
}

// Help sends to user list of available commands
func (b *Bot) Help(chatID chatid) {
	b.Output <- Msg{chatID, funcs}
}

//Run starts all handlers and listeners for bot
func (b *Bot) Run() {

	go b.HandleOutput()
	go b.InputKiller()
	b.SetReminders()

	b.api.Handle("/version", func(m *tba.Message) {
		b.Output <- Msg{chatid(m.Chat.ID), "version 0.4.0"}
	})

	b.api.Handle("/help", func(m *tba.Message) {
		go b.Help(chatid(m.Chat.ID))
	})

	//single line commands don't stop routines
	b.api.Handle("/fiszka", func(m *tba.Message) {
		b.DisplayFlashcard(m)
	})

	b.api.Handle("/dodajfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			//TODO: make it without sleep
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.AddFlashcard(chatID)
	})

	b.api.Handle("/usunfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.DeleteFlashcard(chatID)
	})

	b.api.Handle("/edytujfiszke", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.EditFlashcard(chatID)
	})

	b.api.Handle("/test", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.KnowledgeTest(chatID)
	})

	b.api.Handle("/dodajprzypomnienie", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.AddReminder(chatID)
	})

	b.api.Handle("/pokazprzypomnienia", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)

		go b.ShowReminders(chatID)
	})

	b.api.Handle("/dodajzajecia", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.AddClass(chatID)
	})

	b.api.Handle("/edytujzajecia", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.EditClass(chatID)
	})

	b.api.Handle("/usunzajecia", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.DeleteClass(chatID)
	})

	b.api.Handle("/usunplan", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)
		if _, ok := b.Input[chatID]; ok {
			b.Input[chatID] <- ""
			time.Sleep(2 * time.Second)
		}
		b.Input[chatID] = make(chan string)
		go b.DeleteSchedule(chatID)
	})

	b.api.Handle("/plan", func(m *tba.Message) {
		chatID := chatid(m.Chat.ID)

		go b.ShowSchedule(chatID)
	})

	b.api.Handle(tba.OnText, func(m *tba.Message) {
		if d, ok := b.Input[chatid(m.Chat.ID)]; ok {
			d <- m.Text
		}
	})

	b.api.Start()

}

//NewBot creates new bot instance under given telegram api token. If env is prod it will log all errors and info to a file.
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

	schedules := make(schedulesData)
	_ = ensureDataFileExists(schedulesFileName)
	schedulesData, err := ioutil.ReadFile(schedulesFileName)

	if err != nil {
		log.WithFields(log.Fields{
			"file": schedulesFileName,
		}).Fatal("Could not read file")
	}

	err = json.Unmarshal([]byte(schedulesData), &schedules)

	if err != nil {
		log.WithFields(log.Fields{
			"file": schedulesFileName,
		}).Fatal("Could not decode file")
	}

	input := make(map[chatid]chan string)
	inactiveInput := make(chan chatid)
	output := make(chan Msg)

	log.Info("Bot authorized")
	return &Bot{tb, flashcards, reminders, schedules, input, inactiveInput, output}

}
