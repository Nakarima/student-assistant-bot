package main

import (
	//"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"

	log "github.com/sirupsen/logrus"
	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

const flashcardsFileName = "flashcards.json"

type topic string
type flashcards map[string]string
type flashcardsData map[chatid]map[topic]flashcards

func writeFlashcards(fc flashcardsData, ioLogger *log.Entry) error {
	fcJSON, err := json.Marshal(fc)
	if err != nil {
		ioLogger.Error("Could not encode flashcards")
		return err
	}

	err = ioutil.WriteFile(flashcardsFileName, fcJSON, 0644)
	if err != nil {
		ioLogger.Error("Could not write file")
		return err
	}
	return err
}

func (b *Bot) addFlashcard(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(flashcardsFileName, "addFlashcard")
	defer func() { b.inactiveInput <- chatID }()
	fc := b.flashcardsData

	t, err := b.dialog(chatID, "Podaj temat")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	t = strings.ToLower(t)
	top := topic(t)

	term, err := b.dialog(chatID, "Podaj pojecie")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	term = strings.ToLower(term)
	if _, ok := fc[chatID][top][term]; ok {
		b.output <- msg{chatID, "Fiszka juz istnieje, edytuj za pomoca /edytujfiszke"}
		return
	}

	definition, err := b.dialog(chatID, "Podaj definicje")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	if fc[chatID] == nil {
		fc[chatID] = make(map[topic]flashcards)
	}

	if fc[chatID][top] == nil {
		fc[chatID][top] = make(flashcards)
	}

	fc[chatID][top][term] = definition

	err = writeFlashcards(fc, ioLogger)
	if err != nil {
		b.output <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.flashcardsData[chatID] = fc[chatID]
	b.output <- msg{chatID, "Dodano fiszke"}
}

func (b *Bot) displayFlashcard(m *tba.Message) {

	chatID := chatid(m.Chat.ID)
	fc := b.flashcardsData

	if m.Text == "/fiszka" {
		b.output <- msg{chatID, "Podaj pojecie po spacji"}
		return
	}

	term := strings.ReplaceAll(m.Text, "/fiszka ", "")
	answer := ""

	for top, val := range fc[chatID] {
		if definition, ok := val[strings.ToLower(term)]; ok {
			answer = answer + "\n" + strings.Title(string(top)) + ", " + strings.Title(term) + " - " + definition
		}
	}

	if answer != "" {
		b.output <- msg{
			chatID,
			answer,
		}
		return
	}

	b.output <- msg{chatID, "Nie znaleziono pojecia"}
}

func (b *Bot) deleteFlashcard(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(flashcardsFileName, "deleteFlashcard")
	defer func() { b.inactiveInput <- chatID }()
	fc := b.flashcardsData

	t, err := b.dialog(chatID, "Podaj temat")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	t = strings.ToLower(t)
	top := topic(t)

	term, err := b.dialog(chatID, "Podaj pojecie")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	term = strings.ToLower(term)
	if _, ok := fc[chatID][top][term]; !ok {

		b.output <- msg{chatID, "Fiszka nie istnieje"}
		return
	}

	delete(fc[chatID][top], term)

	if fc[chatID][top] == nil {
		delete(fc[chatID], top)
	}
	err = writeFlashcards(fc, ioLogger)
	if err != nil {
		b.output <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.flashcardsData[chatID] = fc[chatID]
	b.output <- msg{chatID, "Usunieto fiszke"}
}

func (b *Bot) editFlashcard(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(flashcardsFileName, "editFlashcard")
	defer func() { b.inactiveInput <- chatID }()
	fc := b.flashcardsData

	t, err := b.dialog(chatID, "Podaj temat")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	t = strings.ToLower(t)
	top := topic(t)

	term, err := b.dialog(chatID, "Podaj pojecie")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	term = strings.ToLower(term)
	if _, ok := fc[chatID][top][term]; !ok {
		b.output <- msg{chatID, "Fiszka nie istnieje"}
		return
	}

	definition, err := b.dialog(chatID, "Podaj definicje")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	fc[chatID][top][term] = definition
	err = writeFlashcards(fc, ioLogger)
	if err != nil {
		b.output <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.flashcardsData[chatID] = fc[chatID]
	b.output <- msg{chatID, "Edytowano fiszke"}
}
