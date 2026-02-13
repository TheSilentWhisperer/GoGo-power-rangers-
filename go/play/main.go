package main

import (
	"log"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	var app *ui.App = ui.InitializeApp()
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
