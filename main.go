package main

import (
	"os"
)

func main() {
	assistant := NewBot(os.Getenv("telegramBot"), "dev")
	assistant.Run()
}
