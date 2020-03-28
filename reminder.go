package main

import (
	"encoding/json"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
)

const remindersFileName = "reminders.json"
const dateLayout = "02-01-06 15:04"

type reminder struct {
	date  time.Time
	title string
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

	if date.Before(time.Now()) {
		out <- msg{chatID, "Data jest z przeszłości"}
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
