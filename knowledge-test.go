package main

import (
	"strconv"
	"strings"
	log "github.com/sirupsen/logrus"
)

// returns flashcards in given range or if range is equal or higher than number of flashcards returns all
func generateTestFlashcards(fc flashcard, testRange int) flashcard {
	if testRange >= len(fc) {
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

func askQuestions(fc flashcard, chatID chatid, out chan msg, in chan string, state chan chatid, chatLogger *log.Entry) (int, error) {

	correctAnswers := 0
	for term, definition := range fc {
		answer, err := dialog(out, chatID, "Co to jest? "+definition, in)
		if err != nil {
			chatLogger.Info("Dialog ended unsuccessfully")
			state <- chatID
			return 0, err
		}
		answer = strings.ToLower(answer)
		if answer == term {
			out <- msg{chatID, "Poprawna odpowiedz"}
			correctAnswers++
		} else {
			out <- msg{chatID, "Bledna odpowiedz, poprawna to: " + strings.Title(term)}
		}

	}
	return correctAnswers, nil
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
	t = strings.ToLower(t)
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

	testRange, err := strconv.Atoi(testRangeAnswer)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		out <- msg{chatID, "Musisz podac liczbe"}
		state <- chatID
		return
	}

	testFlashcards := generateTestFlashcards(fcTopic, testRange)

	correctAnswers, err := askQuestions(testFlashcards, chatID, out, in, state, chatLogger)
	if err != nil {
		return
	}

	result := "Odpowiedziales poprawnie na " + strconv.Itoa(correctAnswers) + " z " + testRangeAnswer

	out <- msg{chatID, result}

}
