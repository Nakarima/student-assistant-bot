package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
)

const remindersFileName = "reminders.json"
const dateLayout = "02-01-06 15:04"
const remindersTemplate = `
Aktualne przypomnienia:
{{ range . }}{{ .Title }} - {{ .Date.Format "02-01-06 15:04" }}
{{ end }}`

type reminder struct {
	Date  time.Time
	Title string
}

type remindersData map[chatid][]reminder

func writeReminders(rd remindersData, ioLogger *log.Entry) error {
	rdJSON, err := json.Marshal(rd)
	if err != nil {
		ioLogger.Error("Could not encode reminders")
		return err
	}

	err = ioutil.WriteFile(remindersFileName, rdJSON, 0644)
	if err != nil {
		ioLogger.Error("Could not write file")
		return err
	}
	return err
}

func createFromTemplate(rmndrs []reminder) (string, error) {
	tmpl, err := template.New("remindersTemplate").Parse(remindersTemplate)
	if err != nil {
		return "", errors.New("template parse error")
	}

	var answerBuff bytes.Buffer
	err = tmpl.Execute(&answerBuff, rmndrs)
	if err != nil {
		return "", errors.New("template execute error")
	}
	return answerBuff.String(), nil
}

func (b *Bot) showReminders(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	tmpl, err := createFromTemplate(b.remindersData[chatID])
	if err != nil {
		chatLogger.Error("Could not parse reminders")
		return
	}
	b.output <- msg{chatID, tmpl}
}

func (b *Bot) remind(reminder reminder, chatID chatid) {
	select {
	case <-time.After(reminder.Date.Sub(time.Now()) - 26*time.Hour):
		b.output <- msg{chatID, "Przypominam: " + reminder.Title + " " + reminder.Date.Format(dateLayout)}
	}
	select {
	case <-time.After(reminder.Date.Sub(time.Now()) - 2*time.Hour):
		b.output <- msg{chatID, "Przypominam: " + reminder.Title + " " + reminder.Date.Format(dateLayout)}
	}
}

func (b *Bot) setReminders() {
	reminders := b.remindersData
	for chatID, perChatR := range reminders {
		for index, rmndr := range perChatR {
			if rmndr.Date.Before(time.Now().Add(2 * time.Hour)) {
				if index < len(perChatR)-1 {
					copy(perChatR[index:], perChatR[index+1:])
				}
				perChatR[len(perChatR)-1] = reminder{}
				perChatR = perChatR[:len(perChatR)-1]
			} else {
				go b.remind(rmndr, chatID)
			}
		}
		b.remindersData[chatID] = perChatR
	}

	ioLogger := generateIoLogger(remindersFileName, "setReminders")

	writeReminders(reminders, ioLogger)
}

func (b *Bot) addReminder(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(remindersFileName, "addReminder")
	defer func() { b.inactiveInput <- chatID }()
	rd := b.remindersData

	d, err := b.dialog(chatID, "Podaj date w formacie DD-MM-RR HH:MM")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	date, err := time.Parse(dateLayout, d)
	if err != nil {
		b.output <- msg{chatID, "Niepoprawna format daty"}
		return
	}

	if date.Before(time.Now().Add(2 * time.Hour)) {
		b.output <- msg{chatID, "Data jest z przeszłości, spróbuj ponownie"}
		return
	}

	t, err := b.dialog(chatID, "Podaj tytuł przypomnienia")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	rmndr := reminder{date, t}
	go b.remind(rmndr, chatID)

	if rd[chatID] == nil {
		rd[chatID] = []reminder{}
	}

	rd[chatID] = append(rd[chatID], rmndr)

	err = writeReminders(rd, ioLogger)
	if err != nil {
		b.output <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym przypomnieniem w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.remindersData[chatID] = rd[chatID]
	b.output <- msg{chatID, "Dodano przypomnienie"}
}
