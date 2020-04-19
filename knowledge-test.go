package main

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

// returns flashcards in given range or if range is equal or higher than number of flashcards returns all
func generateTestFlashcards(fc flashcards, testRange int) flashcards {
	if testRange >= len(fc) {
		return fc
	}
	testFlashcards := make(flashcards)
	i := 0
	for term, definition := range fc {
		if i > testRange {
			break
		}
		testFlashcards[term] = definition
		i++
	}
	return testFlashcards
}

func (b *Bot) askQuestions(fc flashcards, chatID chatid, chatLogger *log.Entry) (int, error) {

	correctAnswers := 0
	for term, definition := range fc {
		answer, err := b.dialog(chatID, "Co to jest? "+definition)
		if err != nil {
			chatLogger.Info("Dialog ended unsuccessfully")
			return 0, err
		}
		answer = strings.ToLower(answer)
		if answer == term {
			b.output <- msg{chatID, "Poprawna odpowiedz"}
			correctAnswers++
		} else {
			b.output <- msg{chatID, "Bledna odpowiedz, poprawna to: " + strings.Title(term)}
		}

	}
	return correctAnswers, nil
}

func (b *Bot) knowledgeTest(chatID chatid) {

	chatLogger := generateDialogLogger(chatID)
	defer func() { b.inactiveInput <- chatID }()
	fc := b.flashcardsData

	startMessage := "Test twojej wiedzy. Bede podawal definicje roznych pojec, a ty odpowiedz nazwa pojecia. Na poczatek podaj temat, z ktorego chcesz zostac przepytany."

	t, err := b.dialog(chatID, startMessage)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	t = strings.ToLower(t)
	top := topic(t)
	//TODO make inline buttons
	if _, ok := fc[chatID][top]; !ok {
		b.output <- msg{chatID, "Temat nie istnieje"}
		return
	}

	fcTopic := fc[chatID][top]

	askQuestionsNumber := "Podaj ilosc pytan, maksymalna ilosc dla tego tematu: " + strconv.Itoa(len(fcTopic))
	testRangeAnswer, err := b.dialog(chatID, askQuestionsNumber)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	testRange, err := strconv.Atoi(testRangeAnswer)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		b.output <- msg{chatID, "Musisz podac liczbe"}
		return
	}

	testFlashcards := generateTestFlashcards(fcTopic, testRange)

	correctAnswers, err := b.askQuestions(testFlashcards, chatID, chatLogger)
	if err != nil {
		return
	}

	result := "Odpowiedziales poprawnie na " + strconv.Itoa(correctAnswers) + " z " + testRangeAnswer

	b.output <- msg{chatID, result}
}
