package main

import (
	"log"

	fyneapp "fyne.io/fyne/v2/app"
	"github.com/luckygeck/lai/app"
)

func main() {
	log.Println("Starting lai app...")
	a := app.New(fyneapp.New())
	a.Run()
}
