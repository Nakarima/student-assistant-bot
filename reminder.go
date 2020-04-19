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

// Reminder stores info about reminders date and name.
type Reminder struct {
	Date  time.Time
	Title string
}

type remindersData map[chatid][]Reminder

// writeReminders rewrites reminders in .json file. If file doesn't exists it will create a new one.
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

// createFromTemplate creates string with good looking format with info about given reminders.
func createFromTemplate(rmndrs []Reminder) (string, error) {
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

// ShowReminders sends to user all his reminders.
func (b *Bot) ShowReminders(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	tmpl, err := createFromTemplate(b.RemindersData[chatID])
	if err != nil {
		chatLogger.Error("Could not parse reminders")
		return
	}
	b.Output <- Msg{chatID, tmpl}
}

//Remind sends two messages to user with Reminder. One 24 hours(UTC+2, thats why there is 26) before given date, and second one on given Date. After that it will delete reminder from RemindersData.
func (b *Bot) Remind(reminder Reminder, chatID chatid, index int) {
	select {
	case <-time.After(reminder.Date.Sub(time.Now()) - 26*time.Hour):
		b.Output <- Msg{chatID, "Przypominam: " + reminder.Title + " " + reminder.Date.Format(dateLayout)}
	}
	select {
	case <-time.After(reminder.Date.Sub(time.Now()) - 2*time.Hour):
		b.Output <- Msg{chatID, "Przypominam: " + reminder.Title + " " + reminder.Date.Format(dateLayout)}
	}

	rd := b.RemindersData[chatID]
	if index < len(rd)-1 {
		copy(rd[index:], rd[index+1:])
	}
	rd[len(rd)-1] = Reminder{}
	rd = rd[:len(rd)-1]

	b.RemindersData[chatID] = rd
}

// SetReminders is a starter function for setting all reminders after bot startup. It will delete all old reminders.
func (b *Bot) SetReminders() {
	reminders := b.RemindersData
	for chatID, perChatR := range reminders {
		for index, rmndr := range perChatR {
			if rmndr.Date.Before(time.Now().Add(2 * time.Hour)) {
				if index < len(perChatR)-1 {
					copy(perChatR[index:], perChatR[index+1:])
				}
				perChatR[len(perChatR)-1] = Reminder{}
				perChatR = perChatR[:len(perChatR)-1]
			} else {
				go b.Remind(rmndr, chatID, index)
			}
		}
		b.RemindersData[chatID] = perChatR
	}

	ioLogger := generateIoLogger(remindersFileName, "setReminders")

	writeReminders(reminders, ioLogger)
}

// AddReminder starts dialog to create new reminder. Then it will start Remind function and write new reminder to RemindersData.
func (b *Bot) AddReminder(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(remindersFileName, "addReminder")
	defer func() { b.InactiveInput <- chatID }()
	rd := b.RemindersData

	d, err := b.Dialog(chatID, "Podaj date w formacie DD-MM-RR HH:MM")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	date, err := time.Parse(dateLayout, d)
	if err != nil {
		b.Output <- Msg{chatID, "Niepoprawna format daty"}
		return
	}

	if date.Before(time.Now().Add(2 * time.Hour)) {
		b.Output <- Msg{chatID, "Data jest z przeszłości, spróbuj ponownie"}
		return
	}

	t, err := b.Dialog(chatID, "Podaj tytuł przypomnienia")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	rmndr := Reminder{date, t}

	if rd[chatID] == nil {
		rd[chatID] = []Reminder{}
	}

	rd[chatID] = append(rd[chatID], rmndr)
	go b.Remind(rmndr, chatID, len(rd[chatID])-1)

	err = writeReminders(rd, ioLogger)
	if err != nil {
		b.Output <- Msg{chatID, "Wystapil problem, moga wystapic problemy z tym przypomnieniem w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.RemindersData[chatID] = rd[chatID]
	b.Output <- Msg{chatID, "Dodano przypomnienie"}
}
