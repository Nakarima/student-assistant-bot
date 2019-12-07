package main

import (
	//"bytes"
	"encoding/json"
	"io/ioutil"
	"strings"

	log "github.com/sirupsen/logrus"
	tba "gopkg.in/tucnak/telebot.v2" //telegram bot api
)

func addFlashcard(fc flashcards, chatID chatid, out chan msg, in chan string, state chan chatid) {

	chatLogger := log.WithFields(log.Fields{
		"chat": chatID,
	})

	ioLogger := log.WithFields(log.Fields{
		"file": flashcardsFileName,
		"func": "addFlashcard",
	})

	t, err := dialog(out, chatID, "Podaj temat", in)
	if err != nil {

		// nie jestem pewien czy te logi zostawic
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}
	top := topic(t)

	term, err := dialog(out, chatID, "Podaj pojecie", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	if _, ok := fc[chatID][topic(top)][term]; ok {
		out <- msg{chatID, "Fiszka juz istnieje, edytuj za pomoca /edytujfiszke"}
		state <- chatID
		return
	}

	definition, err := dialog(out, chatID, "Podaj definicje", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	if fc[chatID] == nil {
		fc[chatID] = make(map[topic]flashcard)
	}

	if fc[chatID][top] == nil {
		fc[chatID][top] = make(flashcard)
	}

	fc[chatID][top][term] = definition

	fcJSON, err := json.Marshal(fc)
	if err != nil {
		ioLogger.Error("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
		state <- chatID
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, fcJSON, 0644)
	if err != nil {
		ioLogger.Error("Could not write file")
		out <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Dodano fiszke"}
	state <- chatID

}

func displayFlashcard(fc flashcards, m *tba.Message, output chan msg) {

	chatID := chatid(m.Chat.ID)

	if m.Text == "/fiszka" {
		output <- msg{chatID, "Podaj pojecie po spacji"}
		return
	}

	term := strings.ReplaceAll(m.Text, "/fiszka ", "")
	flashcardFound := false

	for top, val := range fc[chatID] {
		if definition, ok := val[term]; ok {
			flashcardFound = true
			output <- msg{
				chatID,
				string(top) + ", " + term + " - " + definition,
			}
		}
	}

	if !flashcardFound {
		output <- msg{chatID, "Nie znaleziono pojecia"}
	}

}

func deleteFlashcard(fc flashcards, chatID chatid, out chan msg, in chan string, state chan chatid) {

	chatLogger := log.WithFields(log.Fields{
		"chat": chatID,
	})

	ioLogger := log.WithFields(log.Fields{
		"file": flashcardsFileName,
		"func": "deleteFlashcard",
	})

	t, err := dialog(out, chatID, "Podaj temat", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}
	top := topic(t)

	term, err := dialog(out, chatID, "Podaj pojecie", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	if _, ok := fc[chatID][top][term]; !ok {

		out <- msg{chatID, "Fiszka nie istnieje"}
		state <- chatID
		return
	}

	delete(fc[chatID][top], term)

	if fc[chatID][top] == nil {
		delete(fc[chatID], top)
	}

	fcJSON, err := json.Marshal(fc)
	if err != nil {
		ioLogger.Error("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
		state <- chatID
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, fcJSON, 0644)
	if err != nil {
		ioLogger.Error("Could not write file")
		out <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Usunieto fiszke"}
	state <- chatID

}

func editFlashcard(fc flashcards, chatID chatid, out chan msg, in chan string, state chan chatid) {

	chatLogger := log.WithFields(log.Fields{
		"chat": chatID,
	})

	ioLogger := log.WithFields(log.Fields{
		"file": flashcardsFileName,
		"func": "editFlashcard",
	})

	t, err := dialog(out, chatID, "Podaj temat", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}
	top := topic(t)

	term, err := dialog(out, chatID, "Podaj pojecie", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	if _, ok := fc[chatID][top][term]; !ok {
		out <- msg{chatID, "Fiszka nie istnieje"}
		state <- chatID
		return
	}

	definition, err := dialog(out, chatID, "Podaj definicje", in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	fc[chatID][top][term] = definition

	fcJSON, err := json.Marshal(fc)
	if err != nil {
		ioLogger.Error("Could not encode flashcards")
		out <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
		state <- chatID
		return
	}

	err = ioutil.WriteFile(flashcardsFileName, fcJSON, 0644)
	if err != nil {
		ioLogger.Error("Could not write file")
		out <- msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
		state <- chatID
		return
	}

	out <- msg{chatID, "Edytowano fiszke"}
	state <- chatID

}
