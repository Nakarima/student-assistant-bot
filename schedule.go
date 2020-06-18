package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
)

const schedulesFileName = "schedules.json"
const timeLayout = "15:04"

// Weekday represents a day
type Weekday int

const (
	monday    Weekday = 1
	tuesday   Weekday = 2
	wednesday Weekday = 3
	thursday  Weekday = 4
	friday    Weekday = 5
	saturday  Weekday = 6
	sunday    Weekday = 7
)
const dayTemplate = `
{{ range . }}{{ .Starts.Format "15:04" }} - {{ .Ends.Format "15:04" }} - {{ .Name }}
{{ end }}`

// Class keeps info about time and name of class.
// Starts defines which hour they start.
// Ends defines which hour they end.
// Name defines name of the class.
type Class struct {
	Starts time.Time
	Ends   time.Time
	Name   string
}

func weekdaysPL() map[string]Weekday {
	return map[string]Weekday{
		"poniedziałek": monday,
		"wtorek":       tuesday,
		"środa":        wednesday,
		"czwartek":     thursday,
		"piątek":       friday,
		"sobota":       saturday,
		"niedziela":    sunday,
	}
}

type schoolDay []Class
type schedule map[Weekday]schoolDay
type schedulesData map[chatid]schedule

// GetString returns polish name of weekday
func (day Weekday) GetString() string {
	names := [...]string{
		"poniedziałek",
		"wtorek",
		"środa",
		"czwartek",
		"piątek",
		"sobota",
		"niedziela",
	}

	if day > sunday || day < monday {
		return "Unknown"
	}

	return names[day-1]
}

func writeSchedule(sd schedulesData, ioLogger *log.Entry) error {
	sdJSON, err := json.Marshal(sd)
	if err != nil {
		ioLogger.Error("Could not encode schedules")
		return err
	}

	err = ioutil.WriteFile(schedulesFileName, sdJSON, 0644)
	if err != nil {
		ioLogger.Error("Could not write file")
		return err
	}
	return err
}

// createDayFromTemplate creates string with good looking format with info about given schedule.
func createDayFromTemplate(day schoolDay, wd Weekday) (string, error) {
	tmpl, err := template.New("dayTemplate").Parse(dayTemplate)
	if err != nil {
		return "", errors.New("template parse error")
	}

	var answerBuff bytes.Buffer
	err = tmpl.Execute(&answerBuff, day)
	if err != nil {
		return "", errors.New("template execute error")
	}

	msg := wd.GetString() + answerBuff.String()
	return msg, nil
}

func createScheduleFromTemplate(sd schedule) (string, error) {
	var msg string
	for d := 0; d < 10; d++ {
		wd := Weekday(d)
		day := sd[wd]
		if len(day) > 0 {
			tmplt, err := createDayFromTemplate(day, wd)
			if err != nil {
				return "", err
			}
			msg = msg + tmplt + "\n"
		}
	}
	return msg, nil
}

func classExists(sd schoolDay, n string) bool {
	for _, cl := range sd {
		if n == cl.Name {
			return true
		}
	}
	return false
}

func findClassByName(sd schoolDay, n string) (Class, error) {
	var c Class
	for _, cl := range sd {
		if n == cl.Name {
			return cl, nil
		}
	}
	return c, errors.New("did not found")
}

func insertClassCorrectly(sd schoolDay, c Class) schoolDay {
	for index, cl := range sd {
		if c.Starts.Before(cl.Starts) {
			begin := sd[:index]
			begin = append(begin, c)
			end := sd[index:]
			return append(begin, end...) // hack because mutabiliti sriti
		}
	}
	return append(sd, c)
}

func deleteClass(sd schoolDay, n string) schoolDay {
	for index, cl := range sd {
		if n == cl.Name {
			begin := sd[:index]
			end := sd[index+1:]
			return append(begin, end...)
		}
	}
	return sd
}

// AddClass launch dialog for creating a new class. It checks if class exists and if not it will add class to schedules and save it in a file
func (b *Bot) AddClass(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(schedulesFileName, "addClass")
	defer func() { b.InactiveInput <- chatID }()

	sd := b.SchedulesData
	w, err := b.Dialog(chatID, "Podaj dzień tygodnia")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	w = strings.ToLower(w)
	wdpl := weekdaysPL()
	if wdpl[w] == 0 {
		b.Output <- Msg{chatID, "Nie znam takiego dnia :("}
		return
	}
	wd := wdpl[w]

	s, err := b.Dialog(chatID, "Podaj godzinę rozpoczęcia w formacie HH:MM")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	start, err := time.Parse(timeLayout, s)
	if err != nil {
		b.Output <- Msg{chatID, "Niepoprawny format godziny"}
		return
	}

	e, err := b.Dialog(chatID, "Podaj godzinę zakończenia w formacie HH:MM")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	end, err := time.Parse(timeLayout, e)
	if err != nil {
		b.Output <- Msg{chatID, "Niepoprawny format godziny"}
		return
	}

	if end.Before(start) {
		b.Output <- Msg{chatID, "Zajęcia nie mogą się kończy przed rozpoczęciem :/"}
		return
	}

	n, err := b.Dialog(chatID, "Podaj nazwę")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	if classExists(sd[chatID][wd], n) {
		b.Output <- Msg{chatID, "Podane zajęcia są juz zapisane"}
		return
	}

	c := Class{start, end, n}

	if sd[chatID] == nil {
		sd[chatID] = schedule{}
	}

	if sd[chatID][wd] == nil {
		sd[chatID][wd] = schoolDay{}
	}

	sd[chatID][wd] = insertClassCorrectly(sd[chatID][wd], c)

	err = writeSchedule(sd, ioLogger)
	if err != nil {
		b.Output <- Msg{chatID, "Wystapil problem, moga wystapic problemy z tymi zajęciami w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.SchedulesData[chatID][wd] = sd[chatID][wd]
	b.Output <- Msg{chatID, "Dodano zajęcia"}
}

// EditClass launch dialog for editing a class. It checks if class exists and if then it will edit class and save it in a file
func (b *Bot) EditClass(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(schedulesFileName, "editClass")
	defer func() { b.InactiveInput <- chatID }()

	sd := b.SchedulesData
	w, err := b.Dialog(chatID, "Podaj dzień tygodnia")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	w = strings.ToLower(w)
	wdpl := weekdaysPL()
	if wdpl[w] == 0 {
		b.Output <- Msg{chatID, "Nie znam takiego dnia :("}
		return
	}
	wd := wdpl[w]

	n, err := b.Dialog(chatID, "Podaj nazwę")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	_, err = findClassByName(sd[chatID][wd], n)
	if err != nil {
		b.Output <- Msg{chatID, "Podane zajęcia nie istnieją"}
		return
	}

	s, err := b.Dialog(chatID, "Podaj godzinę rozpoczęcia w formacie HH:MM")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	start, err := time.Parse(timeLayout, s)
	if err != nil {
		b.Output <- Msg{chatID, "Niepoprawny format godziny"}
		return
	}

	e, err := b.Dialog(chatID, "Podaj godzinę zakończenia w formacie HH:MM")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	end, err := time.Parse(timeLayout, e)
	if err != nil {
		b.Output <- Msg{chatID, "Niepoprawny format godziny"}
		return
	}

	if end.Before(start) {
		b.Output <- Msg{chatID, "Zajęcia nie mogą się kończy przed rozpoczęciem :/"}
		return
	}

	newN, err := b.Dialog(chatID, "Podaj nazwę")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	newC := Class{start, end, newN}

	sd[chatID][wd] = deleteClass(sd[chatID][wd], n)

	sd[chatID][wd] = insertClassCorrectly(sd[chatID][wd], newC)

	err = writeSchedule(sd, ioLogger)
	if err != nil {
		b.Output <- Msg{chatID, "Wystapil problem, moga wystapic problemy z tymi zajęciami w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.SchedulesData[chatID][wd] = sd[chatID][wd]
	b.Output <- Msg{chatID, "Edytowano zajęcia"}
}

// DeleteClass launch dialog for deleting a class. It checks if class exists and if then it will delete class and save it in a file
func (b *Bot) DeleteClass(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(schedulesFileName, "editClass")
	defer func() { b.InactiveInput <- chatID }()

	sd := b.SchedulesData
	w, err := b.Dialog(chatID, "Podaj dzień tygodnia")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	w = strings.ToLower(w)
	wdpl := weekdaysPL()
	if wdpl[w] == 0 {
		b.Output <- Msg{chatID, "Nie znam takiego dnia :("}
		return
	}
	wd := wdpl[w]

	n, err := b.Dialog(chatID, "Podaj nazwę")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	_, err = findClassByName(sd[chatID][wd], n)
	if err != nil {
		b.Output <- Msg{chatID, "Podane zajęcia nie istnieją"}
		return
	}

	sd[chatID][wd] = deleteClass(sd[chatID][wd], n)

	err = writeSchedule(sd, ioLogger)
	if err != nil {
		b.Output <- Msg{chatID, "Wystapil problem, moga wystapic problemy z tymi zajęciami w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.SchedulesData[chatID][wd] = sd[chatID][wd]
	b.Output <- Msg{chatID, "Usunięto zajęcia"}
}

// DeleteSchedule launch dialog for deleting a schedule.
func (b *Bot) DeleteSchedule(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(schedulesFileName, "editClass")
	defer func() { b.InactiveInput <- chatID }()

	sd := b.SchedulesData
	a, err := b.Dialog(chatID, "Napisz 'TAK' zeby usunąć plan")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	if a != "TAK" {
		b.Output <- Msg{chatID, "Ok, nie usuwamy"}
		return
	}

	sd[chatID] = schedule{}

	err = writeSchedule(sd, ioLogger)
	if err != nil {
		b.Output <- Msg{chatID, "Wystapil problem, moga wystapic problemy z tymi zajęciami w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.SchedulesData[chatID] = sd[chatID]
	b.Output <- Msg{chatID, "Usunięto plan"}
}

// ShowSchedule sends to user schedule
func (b *Bot) ShowSchedule(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	tmpl, err := createScheduleFromTemplate(b.SchedulesData[chatID])
	if err != nil {
		chatLogger.Error("Could not parse schedule")
		return
	}
	if tmpl == "" {
		tmpl = "Brak zajęć :/"
	}
	b.Output <- Msg{chatID, tmpl}
}
