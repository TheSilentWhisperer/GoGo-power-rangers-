package ui

import "github.com/hajimehoshi/ebiten/v2"

type KeyState struct {
	Key        ebiten.Key
	WasPressed bool
	IsPressed  bool
}

func NewKeyState(key ebiten.Key) *KeyState {
	return &KeyState{
		Key:        key,
		WasPressed: false,
		IsPressed:  false,
	}
}

func (ks *KeyState) Update() {
	ks.WasPressed = ks.IsPressed
	ks.IsPressed = ebiten.IsKeyPressed(ks.Key)
}

func (ks *KeyState) JustPressed() bool {
	return ks.IsPressed && !ks.WasPressed
}

type InputState interface {
	Update()
	JustPressed() bool
	IsPressed() bool
}
