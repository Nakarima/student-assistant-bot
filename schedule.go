package main

import (
	"strings"
	"time"
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

// Class keeps info about time and name of class.
type Class struct {
	starts time.Time
	ends   time.Time
	name   string
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

	if day < Sunday || day > Saturday {
		return "Unknown"
	}

	return names[day-1]
}

func classExists(sd schoolDay, c Class) bool {
	for cl := range sd {
		if c == cl {
			return false
		}
	}
	return true
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
		b.Output <- Msg{chatID, "Taki dzień nie istnieje"}
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

	n, err := b.Dialog(chatID, "Podaj nazwę")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	c := Class{start, end, n}

	if classExists(sd[chatID][wd], c) {
		b.Output <- Msg{chatID, "Podane zajęcia są juz zapisane"}
		return
	}

	if sd[chatID] == nil {
		sd[chatID] = schedule{}
	}

	if sd[chatID][wd] == nil {
		sd[chatID][wd] = schoolDay{}
	}

	sd[chatID][wd] = append(sd[chatID][wd], c)

}
