package main

import (
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"
)

// generateTestFlashcards returns flashcards in given range or if range is equal or higher than number of flashcards returns all flashcards
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

// AskQuestions starts dialog in which bot sends definitions and user has to answer with correct term. It returns sum of correct answers.
func (b *Bot) AskQuestions(fc flashcards, chatID chatid, chatLogger *log.Entry) (int, error) {
	correctAnswers := 0
	for term, definition := range fc {
		answer, err := b.Dialog(chatID, "Co to jest? "+definition)
		if err != nil {
			chatLogger.Info("Dialog ended unsuccessfully")
			return 0, err
		}
		answer = strings.ToLower(answer)
		if answer == term {
			b.Output <- Msg{chatID, "Poprawna odpowiedz"}
			correctAnswers++
		} else {
			b.Output <- Msg{chatID, "Bledna odpowiedz, poprawna to: " + strings.Title(term)}
		}
	}

	return correctAnswers, nil
}

// KnowledgeTest starts dialog in which it asks for topic of flashcards and number of questions. Then it starts AskQuestions. After that it sends to user his score.
func (b *Bot) KnowledgeTest(chatID chatid) {
	chatLogger := generateDialogLogger(chatID)
	defer func() { b.InactiveInput <- chatID }()
	fc := b.FlashcardsData

	startMessage := "Test twojej wiedzy. Bede podawal definicje roznych pojec, a ty odpowiedz nazwa pojecia. Na poczatek podaj temat, z ktorego chcesz zostac przepytany."

	t, err := b.Dialog(chatID, startMessage)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}
	t = strings.ToLower(t)
	top := topic(t)
	//TODO make inline buttons
	if _, ok := fc[chatID][top]; !ok {
		b.Output <- Msg{chatID, "Temat nie istnieje"}
		return
	}

	fcTopic := fc[chatID][top]

	askQuestionsNumber := "Podaj ilosc pytan, maksymalna ilosc dla tego tematu: " + strconv.Itoa(len(fcTopic))
	testRangeAnswer, err := b.Dialog(chatID, askQuestionsNumber)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		return
	}

	testRange, err := strconv.Atoi(testRangeAnswer)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		b.Output <- Msg{chatID, "Musisz podac liczbe"}
		return
	}

	testFlashcards := generateTestFlashcards(fcTopic, testRange)

	correctAnswers, err := b.AskQuestions(testFlashcards, chatID, chatLogger)
	if err != nil {
		return
	}

	result := "Odpowiedziales poprawnie na " + strconv.Itoa(correctAnswers) + " z " + testRangeAnswer

	b.Output <- Msg{chatID, result}
}
