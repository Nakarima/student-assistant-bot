package main

import (
	"bytes"
	"errors"
	"encoding/json"
	"io/ioutil"
	"time"
	"text/template"

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

type reminders map[chatid][]reminder

func writeReminders(rd reminders, ioLogger *log.Entry) error {
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

func createFromTemplate(rmndrs []reminder, ) (string, error) {
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

func showReminders(reminders reminders, chatID chatid, out chan msg) {
	chatLogger := generateDialogLogger(chatID)
	tmpl, err := createFromTemplate(reminders[chatID])
	if err != nil {
		chatLogger.Error("Could not parse reminders")
		return
	}
	out <- msg{chatID, tmpl}
}


func remind(reminder reminder, chatID chatid, out chan msg) {
	select {
	case <-time.After(reminder.Date.Sub(time.Now()) - 26*time.Hour):
		out <- msg{chatID, "Przypominam: " + reminder.Title + " " + reminder.Date.Format(dateLayout)}
	}
	select {
	case <-time.After(reminder.Date.Sub(time.Now()) - 2*time.Hour):
		out <- msg{chatID, "Przypominam: " + reminder.Title + " " + reminder.Date.Format(dateLayout)}
	}
}

func setReminders(reminders reminders, out chan msg) {
	for chatID, perChat := range reminders {
		for index, rmndr := range perChat {
			if rmndr.Date.Before(time.Now().Add(2*time.Hour)) {
				if index < len(perChat)-1 {
					copy(perChat[index:], perChat[index+1:])
				}
				perChat[len(perChat)-1] = reminder{}
				perChat = perChat[:len(perChat)-1]
			} else {
				go remind(rmndr, chatID, out)
			}
		}
	}

	ioLogger := generateIoLogger(remindersFileName, "setReminders")

	writeReminders(reminders, ioLogger)
}

func addReminder(rd reminders, chatID chatid, out chan msg, in chan string, state chan chatid) {
	chatLogger := generateDialogLogger(chatID)

	ioLogger := generateIoLogger(remindersFileName, "addReminder")
	d, err := dialog(out, chatID, "Podaj date w formacie DD-MM-RR HH:MM", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}
	date, err := time.Parse(dateLayout, d)
	if err != nil {
		out <- msg{chatID, "Niepoprawna format daty"}
		state <- chatID
		return
	}

	if date.Before(time.Now().Add(2*time.Hour)) {
		out <- msg{chatID, "Data jest z przeszłości, spróbuj ponownie"}
		state <- chatID
		return
	}

	t, err := dialog(out, chatID, "Podaj tytuł przypomnienia", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	rmndr := reminder{date, t}
	go remind(rmndr, chatID, out)

	if rd[chatID] == nil {
		rd[chatID] = []reminder{}
	}

	rd[chatID] = append(rd[chatID], rmndr)

	err = writeReminders(rd, ioLogger)
	if err != nil {
		out <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym przypomnieniem w przyszlosci, skontaktuj sie z administratorem"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Dodano przypomnienie"}
	state <- chatID
}
