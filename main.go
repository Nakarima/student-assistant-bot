package main

import (
	"os"
)

func main() {
	assistant := NewBot(os.Getenv("telegramBot"))
	assistant.Run()
}
