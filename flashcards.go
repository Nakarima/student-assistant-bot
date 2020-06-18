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

// writeFlashcards rewrites flashcards in .json file. If file doesn't exists it will create a new one.
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

// AddFlashcard launch dialog for creating a new flashcard. It checks if flashcard exists and if not it will add flashcards to FlashcardsData and save it in a file.
func (b *Bot) AddFlashcard(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(flashcardsFileName, "addFlashcard")
	fc := b.FlashcardsData
	defer func() { b.InactiveInput <- chatID }()

	t, err := b.Dialog(chatID, "Podaj temat")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	t = strings.ToLower(t)
	top := topic(t)

	term, err := b.Dialog(chatID, "Podaj pojecie")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	term = strings.ToLower(term)
	if _, ok := fc[chatID][top][term]; ok {
		b.Output <- Msg{chatID, "Fiszka juz istnieje, edytuj za pomoca /edytujfiszke"}
		return
	}

	definition, err := b.Dialog(chatID, "Podaj definicje")
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
		b.Output <- Msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.FlashcardsData[chatID] = fc[chatID]
	b.Output <- Msg{chatID, "Dodano fiszke"}
}

// DisplayFlashcard searches FlashcardsData for given term and sends defintion to user if finds it.
func (b *Bot) DisplayFlashcard(m *tba.Message) {
	chatID := chatid(m.Chat.ID)
	fc := b.FlashcardsData

	if m.Text == "/fiszka" {
		b.Output <- Msg{chatID, "Podaj pojecie po spacji"}
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
		b.Output <- Msg{
			chatID,
			answer,
		}
		return
	}

	b.Output <- Msg{chatID, "Nie znaleziono pojecia"}
}

// DeleteFlashcard starts dialog with user to check if given flashcard exists. If it exists, it will be deleted from FlashcardData.
func (b *Bot) DeleteFlashcard(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(flashcardsFileName, "deleteFlashcard")
	defer func() { b.InactiveInput <- chatID }()
	fc := b.FlashcardsData

	t, err := b.Dialog(chatID, "Podaj temat")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	t = strings.ToLower(t)
	top := topic(t)

	term, err := b.Dialog(chatID, "Podaj pojecie")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	term = strings.ToLower(term)
	if _, ok := fc[chatID][top][term]; !ok {

		b.Output <- Msg{chatID, "Fiszka nie istnieje"}
		return
	}

	delete(fc[chatID][top], term)

	if fc[chatID][top] == nil {
		delete(fc[chatID], top)
	}
	err = writeFlashcards(fc, ioLogger)
	if err != nil {
		b.Output <- Msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.FlashcardsData[chatID] = fc[chatID]
	b.Output <- Msg{chatID, "Usunieto fiszke"}
}

// EditFlashcard starts dialog with user to check if given flashcard exists. If it exists, it's definition is edited and saved in FlashcardsData.
func (b *Bot) EditFlashcard(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	ioLogger := generateIoLogger(flashcardsFileName, "editFlashcard")
	defer func() { b.InactiveInput <- chatID }()
	fc := b.FlashcardsData

	t, err := b.Dialog(chatID, "Podaj temat")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	t = strings.ToLower(t)
	top := topic(t)

	term, err := b.Dialog(chatID, "Podaj pojecie")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	term = strings.ToLower(term)
	if _, ok := fc[chatID][top][term]; !ok {
		b.Output <- Msg{chatID, "Fiszka nie istnieje"}
		return
	}

	definition, err := b.Dialog(chatID, "Podaj definicje")
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	fc[chatID][top][term] = definition
	err = writeFlashcards(fc, ioLogger)
	if err != nil {
		b.Output <- Msg{chatID, "Wystapil problem, moga wystapic problemy z tym terminem w przyszlosci, skontaktuj sie z administratorem"}
	}

	b.FlashcardsData[chatID] = fc[chatID]
	b.Output <- Msg{chatID, "Edytowano fiszke"}
}
