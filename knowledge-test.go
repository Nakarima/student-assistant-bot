package main

import (
	"strconv"

	log "github.com/sirupsen/logrus"
)

func knowledgeTest(fc flashcards, chatID chatid, out chan msg, in chan string, state chan chatid) {

	chatLogger := log.WithFields(log.Fields{
		"chat": chatID,
	})

	startMessage := "Test twojej wiedzy. Bede podawal definicje roznych pojec, a ty odpowiedz nazwa pojecia. Na poczatek podaj temat, z ktorego chcesz zostac przepytany."

	t, err := dialog(out, chatID, startMessage, in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}
	top := topic(t)
	if _, ok := fc[chatID][top]; !ok {
		out <- msg{chatID, "Temat nie istnieje"}
		state <- chatID
		return
	}

	testFlashcards := fc[chatID][top]

	flashcardsLength := len(testFlashcards)

	testRangeAnswer, err := dialog(out, chatID, "Podaj ilosc pytan, maksymalna ilosc dla tego tematu: "+strconv.Itoa(flashcardsLength), in)
	if err != nil {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	testRange, err2 := strconv.Atoi(testRangeAnswer)
	i := 0
	for err2 != nil && i < 3 {
		testRangeAnswer, err = dialog(out, chatID, "Musisz podac liczbe", in)
		if err != nil {
			chatLogger.Info("Dialog ended unsuccessfully")
			state <- chatID
			return
		}
		testRange, err2 = strconv.Atoi(testRangeAnswer)
		i++
	}
	if i >= 3 {
		chatLogger.Info("Dialog ended unsuccessfully")
		state <- chatID
		return
	}

	if testRange > flashcardsLength {
		testRange = flashcardsLength
	}

	i = 0
	correctAnswers := 0
	for term, definition := range testFlashcards {
		if i >= testRange {
			break
		}

		answer, err := dialog(out, chatID, "Co to jest? "+definition, in)
		if err != nil {
			chatLogger.Info("Dialog ended unsuccessfully")
			state <- chatID
			return
		}

		if answer == term {
			out <- msg{chatID, "Poprawna odpowiedz"}
			correctAnswers++
		} else {
			out <- msg{chatID, "Bledna odpowiedz, poprawna to: " + term}
		}

		i++
	}

	result := "Odpowiedziales poprawnie na " + strconv.Itoa(correctAnswers) + " z " + testRangeAnswer

	out <- msg{chatID, result}

}
