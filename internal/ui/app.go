package ui

import (
	"image/color"
	"sync"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type LockedBool struct {
	mu    sync.Mutex
	value bool
}

func NewLockedBool(value bool) *LockedBool {
	return &LockedBool{
		value: value,
	}
}

func (lb *LockedBool) get() bool {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	return lb.value
}

func (lb *LockedBool) set(value bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.value = value
}

type UIMetadata struct {
	WindowTitle                         string
	Margin                              float32
	BoardSize                           float32
	HighlightedIntersectionsRadiusScale float32
	StoneRadiusScale                    float32
}

func NewUI(windowTitle string, margin, boardSize, highlightedIntersectionsRadiusScale, stoneRadiusScale float32) *UIMetadata {
	return &UIMetadata{
		WindowTitle:                         windowTitle,
		Margin:                              margin,
		BoardSize:                           boardSize,
		HighlightedIntersectionsRadiusScale: highlightedIntersectionsRadiusScale,
		StoneRadiusScale:                    stoneRadiusScale,
	}
}

type App struct {
	isThinking *LockedBool
	BlackAgent agents.Agent
	WhiteAgent agents.Agent
	Game       *environment.Game
	UIMetadata *UIMetadata
}

func NewApp(blackAgent, whiteAgent agents.Agent, game *environment.Game, ui_metadata *UIMetadata) *App {
	return &App{
		isThinking: NewLockedBool(false),
		BlackAgent: blackAgent,
		WhiteAgent: whiteAgent,
		Game:       game,
		UIMetadata: ui_metadata,
	}
}

func (app *App) DrawBackground(ebitenImage *ebiten.Image) {
	// Background
	ebitenImage.Fill(color.RGBA{200, 170, 120, 255})
}

func (app *App) CellSize() float32 {
	return app.UIMetadata.BoardSize / float32(app.Game.Board.Width-1)
}

func (app *App) windowHeight() int {
	return int(2*app.UIMetadata.Margin + app.UIMetadata.BoardSize)
}

func (app *App) windowWidth() int {
	return int(2*app.UIMetadata.Margin + app.UIMetadata.BoardSize)
}

func (app *App) highlightedIntersections(ebitenImage *ebiten.Image) []environment.Position {
	var highlightedPositions []environment.Position = make([]environment.Position, 0, 9)
	switch app.Game.Board.Height {
	case 9:
		highlightedPositions = []environment.Position{
			environment.NewPosition(2, 2),
			environment.NewPosition(2, 4),
			environment.NewPosition(2, 6),
			environment.NewPosition(4, 2),
			environment.NewPosition(4, 4),
			environment.NewPosition(4, 6),
			environment.NewPosition(6, 2),
			environment.NewPosition(6, 4),
			environment.NewPosition(6, 6),
		}
	default:
		panic("highlightedIntersections: unsupported board size")
	}
	return highlightedPositions
}

func (app *App) DrawGrid(ebitenImage *ebiten.Image) {

	const strokeWidth float32 = 2
	var lineColor color.Color = color.Black
	const antialias bool = true

	// Draw horizontal lines
	for i := 0; i < app.Game.Board.Height; i++ {
		var x0, y0, x1, y1 float32 = app.UIMetadata.Margin, app.UIMetadata.Margin + app.CellSize()*float32(i), app.UIMetadata.BoardSize + app.UIMetadata.Margin, app.UIMetadata.Margin + app.CellSize()*float32(i)
		vector.StrokeLine(ebitenImage, x0, y0, x1, y1, strokeWidth, lineColor, antialias)
	}

	// Draw vertical lines
	for j := 0; j < app.Game.Board.Width; j++ {
		var x0, y0, x1, y1 float32 = app.UIMetadata.Margin + app.CellSize()*float32(j), app.UIMetadata.Margin, app.UIMetadata.Margin + app.CellSize()*float32(j), app.UIMetadata.BoardSize + app.UIMetadata.Margin
		vector.StrokeLine(ebitenImage, x0, y0, x1, y1, strokeWidth, lineColor, antialias)
	}

	// Draw highlighted intersections
	var highlightedPositions []environment.Position = app.highlightedIntersections(ebitenImage)
	for _, pos := range highlightedPositions {
		var cx, cy float32 = app.UIMetadata.Margin + app.CellSize()*float32(pos.J), app.UIMetadata.Margin + app.CellSize()*float32(pos.I)
		var radius float32 = app.CellSize() * app.UIMetadata.HighlightedIntersectionsRadiusScale
		vector.FillCircle(ebitenImage, cx, cy, radius, lineColor, antialias)
	}
}

func (app *App) DrawStones(ebitenImage *ebiten.Image) {
	const antialias bool = true
	for i := 0; i < app.Game.Board.Height; i++ {
		for j := 0; j < app.Game.Board.Width; j++ {
			var stone environment.Stone = app.Game.Board.Matrix[i][j]
			if stone == environment.Empty {
				continue
			}
			var cx, cy float32 = app.UIMetadata.Margin + app.CellSize()*float32(j), app.UIMetadata.Margin + app.CellSize()*float32(i)
			var radius float32 = app.CellSize() * app.UIMetadata.StoneRadiusScale
			var fillColor color.Color
			switch stone {
			case environment.Black:
				fillColor = color.Black
			case environment.White:
				fillColor = color.White
			}
			vector.FillCircle(ebitenImage, cx, cy, radius, fillColor, antialias)
		}
	}
}

func (app *App) Draw(ebitenImage *ebiten.Image) {
	app.DrawBackground(ebitenImage)
	app.DrawGrid(ebitenImage)
	app.DrawStones(ebitenImage)

}

func (app *App) Update() error {
	if app.Game.IsTerminal() {
		return nil // Game over, no more updates
	}

	var currentAgent agents.Agent
	if app.Game.Board.CurrentPlayer == environment.Black {
		currentAgent = app.BlackAgent
	} else {
		currentAgent = app.WhiteAgent
	}

	if app.isThinking.get() {
		return nil // Still thinking, wait for the next update
	}

	go func() {
		app.isThinking.set(true)
		var gameCopy *environment.Game = app.Game.DeepCopy()
		var action environment.Action = currentAgent.SelectAction(gameCopy)
		app.Game.PlayAction(action)
		app.isThinking.set(false)
	}()

	return nil
}

func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return app.windowWidth(), app.windowHeight()
}

func InitializeApp() *App {
	var blackAgent agents.Agent = agents.NewMCTSAgent(10000, 8, -1.0)
	var whiteAgent agents.Agent = agents.NewMCTSAgent(10000, 8, -1.0)
	var game *environment.Game = environment.NewGame(
		9,   // height
		9,   // width
		6.5, // komi
	)

	const margin float32 = 40
	const BoardSize float32 = 400
	const WindowTitle string = "Go Game"
	const HighlightedIntersectionsRadiusScale float32 = 0.1
	const StoneRadiusScale float32 = 0.4

	var ui_metadata *UIMetadata = NewUI(WindowTitle, margin, BoardSize, HighlightedIntersectionsRadiusScale, StoneRadiusScale)
	var app *App = NewApp(blackAgent, whiteAgent, game, ui_metadata)

	ebiten.SetWindowSize(app.windowWidth(), app.windowHeight())
	ebiten.SetWindowTitle(WindowTitle)
	return app
}
