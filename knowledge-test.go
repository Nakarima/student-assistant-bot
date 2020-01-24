package main

import (
	"strconv"

	log "github.com/sirupsen/logrus"
)

func generateTestFlashcards(fc flashcard, testRange int) flashcard {
	if testRange > len(fc) {
		return fc
	}
	testFlashcards := make(flashcard)
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

func askQuestions(fc flashcard, chatID chatid, out chan msg, in chan string, state chan chatid, chatLogger *log.Entry) int {

	correctAnswers := 0
	for term, definition := range fc {
		answer, err := dialog(out, chatID, "Co to jest? "+definition, in)
		if err != nil {
			chatLogger.Info("Dialog ended unsuccessfully")
			state <- chatID
			return 0
		}

		if answer == term {
			out <- msg{chatID, "Poprawna odpowiedz"}
			correctAnswers++
		} else {
			out <- msg{chatID, "Bledna odpowiedz, poprawna to: " + term}
		}

	}
	return correctAnswers
}

func knowledgeTest(fc flashcards, chatID chatid, out chan msg, in chan string, state chan chatid) {

	chatLogger := generateDialogLogger(chatID)

	startMessage := "Test twojej wiedzy. Bede podawal definicje roznych pojec, a ty odpowiedz nazwa pojecia. Na poczatek podaj temat, z ktorego chcesz zostac przepytany."

	t, err := dialog(out, chatID, startMessage, in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}
	top := topic(t)
	//TODO make inline buttons
	if _, ok := fc[chatID][top]; !ok {
		out <- msg{chatID, "Temat nie istnieje"}
		state <- chatID
		return
	}

	fcTopic := fc[chatID][top]

	testRangeAnswer, err := dialog(out, chatID, "Podaj ilosc pytan, maksymalna ilosc dla tego tematu: "+strconv.Itoa(len(fcTopic)), in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	testRange, err2 := strconv.Atoi(testRangeAnswer)
	if err2 != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		out <- msg{chatID, "Musisz podac liczbe"}
		state <- chatID
		return
	}

	testFlashcards := generateTestFlashcards(fcTopic, testRange)

	correctAnswers := askQuestions(testFlashcards, chatID, out, in, state, chatLogger)

	result := "Odpowiedziales poprawnie na " + strconv.Itoa(correctAnswers) + " z " + testRangeAnswer

	out <- msg{chatID, result}

}
