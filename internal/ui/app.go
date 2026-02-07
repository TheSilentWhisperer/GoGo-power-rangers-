package ui

import (
	"image/color"
	"sync"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
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

type LockedGame struct {
	mu   sync.Mutex
	game *environment.Game
}

func NewLockedGame(game *environment.Game) *LockedGame {
	return &LockedGame{
		game: game,
	}
}

func (lg *LockedGame) get() *environment.Game {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	return lg.game
}

func (lg *LockedGame) set(game *environment.Game) {
	lg.mu.Lock()
	defer lg.mu.Unlock()
	lg.game = game
}

type Margin struct {
	Top    float32
	Bottom float32
	Left   float32
	Right  float32
}

func NewMargin(top, bottom, left, right float32) Margin {
	return Margin{
		Top:    top,
		Bottom: bottom,
		Left:   left,
		Right:  right,
	}
}

type UIMetadata struct {
	WindowTitle                         string
	Margin                              Margin
	BoardSize                           float32
	HighlightedIntersectionsRadiusScale float32
	StoneRadiusScale                    float32
	DescriptionBarHeight                float32
	PassSquareSizeScale                 float32
}

func NewUI(windowTitle string, margin Margin, boardSize, highlightedIntersectionsRadiusScale, stoneRadiusScale, descriptionBarHeight, passSquareSizeScale float32) *UIMetadata {
	return &UIMetadata{
		WindowTitle:                         windowTitle,
		Margin:                              margin,
		BoardSize:                           boardSize,
		HighlightedIntersectionsRadiusScale: highlightedIntersectionsRadiusScale,
		StoneRadiusScale:                    stoneRadiusScale,
		DescriptionBarHeight:                descriptionBarHeight,
		PassSquareSizeScale:                 passSquareSizeScale,
	}
}

type App struct {
	isThinking *LockedBool
	BlackAgent agents.Agent
	WhiteAgent agents.Agent
	Game       *LockedGame
	UIMetadata *UIMetadata
}

func NewApp(blackAgent, whiteAgent agents.Agent, game *environment.Game, ui_metadata *UIMetadata) *App {
	return &App{
		isThinking: NewLockedBool(false),
		BlackAgent: blackAgent,
		WhiteAgent: whiteAgent,
		Game:       NewLockedGame(game),
		UIMetadata: ui_metadata,
	}
}

func (app *App) DrawBackground(ebitenImage *ebiten.Image) {
	// Background
	ebitenImage.Fill(color.RGBA{200, 170, 120, 255})
}

func (app *App) CellSize() float32 {
	return app.UIMetadata.BoardSize / float32(app.Game.get().Board.Width-1)
}

func (app *App) windowHeight() int {
	return int(app.UIMetadata.Margin.Top + app.UIMetadata.BoardSize + app.UIMetadata.Margin.Bottom + app.UIMetadata.DescriptionBarHeight)
}

func (app *App) windowWidth() int {
	return int(app.UIMetadata.Margin.Left + app.UIMetadata.BoardSize + app.UIMetadata.Margin.Right)
}

func (app *App) highlightedIntersections(ebitenImage *ebiten.Image) []environment.Position {
	var highlightedPositions []environment.Position = make([]environment.Position, 0, 9)
	switch app.Game.get().Board.Height {
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
	for i := 0; i < app.Game.get().Board.Height; i++ {
		var x0, y0, x1, y1 float32 = app.UIMetadata.Margin.Left, app.UIMetadata.Margin.Top + app.CellSize()*float32(i), app.UIMetadata.BoardSize + app.UIMetadata.Margin.Left, app.UIMetadata.Margin.Top + app.CellSize()*float32(i)
		vector.StrokeLine(ebitenImage, x0, y0, x1, y1, strokeWidth, lineColor, antialias)
	}

	// Draw vertical lines
	for j := 0; j < app.Game.get().Board.Width; j++ {
		var x0, y0, x1, y1 float32 = app.UIMetadata.Margin.Left + app.CellSize()*float32(j), app.UIMetadata.Margin.Top, app.UIMetadata.Margin.Left + app.CellSize()*float32(j), app.UIMetadata.BoardSize + app.UIMetadata.Margin.Top
		vector.StrokeLine(ebitenImage, x0, y0, x1, y1, strokeWidth, lineColor, antialias)
	}

	// Draw highlighted intersections
	var highlightedPositions []environment.Position = app.highlightedIntersections(ebitenImage)
	for _, pos := range highlightedPositions {
		var cx, cy float32 = app.UIMetadata.Margin.Left + app.CellSize()*float32(pos.J), app.UIMetadata.Margin.Top + app.CellSize()*float32(pos.I)
		var radius float32 = app.CellSize() * app.UIMetadata.HighlightedIntersectionsRadiusScale
		vector.FillCircle(ebitenImage, cx, cy, radius, lineColor, antialias)
	}
}

func (app *App) DrawStones(ebitenImage *ebiten.Image) {
	const antialias bool = true
	for i := 0; i < app.Game.get().Board.Height; i++ {
		for j := 0; j < app.Game.get().Board.Width; j++ {
			var stone environment.Stone = app.Game.get().Board.Matrix[i][j]
			if stone == environment.Empty {
				continue
			}
			var cx, cy float32 = app.UIMetadata.Margin.Left + app.CellSize()*float32(j), app.UIMetadata.Margin.Top + app.CellSize()*float32(i)
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

func (app *App) PassSquareMarginScale() float32 {
	return 0.5 * (1 - app.UIMetadata.PassSquareSizeScale)
}

func (app *App) PassSquareMargin() float32 {
	return app.PassSquareMarginScale() * app.UIMetadata.DescriptionBarHeight
}

func (app *App) PassSquareSize() float32 {
	return app.UIMetadata.PassSquareSizeScale * app.UIMetadata.DescriptionBarHeight
}

func (app *App) PassSquarePosition(player environment.Stone) (float32, float32) {
	switch player {
	case environment.Black:
		//top left corner of the square
		return app.UIMetadata.Margin.Left + app.UIMetadata.BoardSize - (3*app.PassSquareMargin() + 2*app.PassSquareSize()), app.UIMetadata.Margin.Top + app.UIMetadata.BoardSize + app.UIMetadata.Margin.Bottom + app.PassSquareMargin()
	case environment.White:
		return app.UIMetadata.Margin.Left + app.UIMetadata.BoardSize - (app.PassSquareMargin() + app.PassSquareSize()), app.UIMetadata.Margin.Top + app.UIMetadata.BoardSize + app.UIMetadata.Margin.Bottom + app.PassSquareMargin()
	default:
		panic("PassSquarePosition: invalid player")
	}
}

func (app *App) DescriptionBarWidth() float32 {
	return app.UIMetadata.Margin.Left + app.UIMetadata.BoardSize + app.UIMetadata.Margin.Right - 2*app.UIMetadata.DescriptionBarHeight // We need to leave space for the pass squares
}

func (app *App) DescriptionBarCenter() (float64, float64) {
	return float64(app.DescriptionBarWidth()) / 2, float64(app.UIMetadata.Margin.Top) + float64(app.UIMetadata.BoardSize) + float64(app.UIMetadata.Margin.Bottom) + float64(app.UIMetadata.DescriptionBarHeight)/2
}

func (app *App) DrawDescriptionBar(ebitenImage *ebiten.Image) {

	// Write in the bottom margin
	const antialias bool = true
	var fontFace font.Face = basicfont.Face7x13
	var textFace text.Face = text.NewGoXFace(fontFace)
	var drawOptions *text.DrawOptions = &text.DrawOptions{}
	var textCenterX, textCenterY float64 = app.DescriptionBarCenter()
	drawOptions.GeoM.Translate(textCenterX, textCenterY)
	drawOptions.PrimaryAlign = text.AlignCenter
	drawOptions.SecondaryAlign = text.AlignCenter

	// Draw description text
	var descriptionText string
	if app.Game.get().IsTerminal() {
		var winner environment.Stone = app.Game.get().GetWinner()
		switch winner {
		case environment.Empty:
			descriptionText = "Game over: Draw"
		case environment.Black:
			descriptionText = "Game over: Black wins"
		case environment.White:
			descriptionText = "Game over: White wins"
		}
	} else {
		switch app.Game.get().Board.CurrentPlayer {
		case environment.Black:
			descriptionText = "Black"
		case environment.White:
			descriptionText = "White"
		}
		switch app.isThinking.get() {
		case true:
			descriptionText += " is thinking..."
		default:
			descriptionText += " is ready to play!"
		}
	}
	text.Draw(ebitenImage, descriptionText, textFace, drawOptions)
}

func (app *App) DrawPassSquare(ebitenImage *ebiten.Image) {

	var squareSize float32 = app.PassSquareSize()
	// background colors for the pass squares
	var notPassedBackgroundColor color.Color = color.RGBA{255, 0, 0, 50}
	var PassedBackgroundColor color.Color = color.RGBA{0, 255, 0, 50}

	//draw square for black
	var x, y float32 = app.PassSquarePosition(environment.Black)
	if app.Game.get().Board.Passes.Black {
		vector.FillRect(ebitenImage, x, y, squareSize, squareSize, PassedBackgroundColor, true)
	} else {
		vector.FillRect(ebitenImage, x, y, squareSize, squareSize, notPassedBackgroundColor, true)
	}
	vector.StrokeRect(ebitenImage, x, y, squareSize, squareSize, 2, color.Black, true)

	//draw square for white
	x, y = app.PassSquarePosition(environment.White)
	if app.Game.get().Board.Passes.White {
		vector.FillRect(ebitenImage, x, y, squareSize, squareSize, PassedBackgroundColor, true)
	} else {
		vector.FillRect(ebitenImage, x, y, squareSize, squareSize, notPassedBackgroundColor, true)
	}
	vector.StrokeRect(ebitenImage, x, y, squareSize, squareSize, 2, color.White, true)
}

func (app *App) Draw(ebitenImage *ebiten.Image) {
	app.DrawBackground(ebitenImage)
	app.DrawGrid(ebitenImage)
	app.DrawStones(ebitenImage)
	app.DrawDescriptionBar(ebitenImage)
	app.DrawPassSquare(ebitenImage)
}

func (app *App) Update() error {

	if app.Game.get().IsTerminal() {
		return nil // Game over, no more updates needed
	}

	var currentAgent agents.Agent
	if app.Game.get().Board.CurrentPlayer == environment.Black {
		currentAgent = app.BlackAgent
	} else {
		currentAgent = app.WhiteAgent
	}

	if app.isThinking.get() {
		return nil // Still thinking, wait for the next update
	}

	go func() {
		app.isThinking.set(true)
		var gameCopy *environment.Game = app.Game.get().DeepCopy()
		var action environment.Action = currentAgent.SelectAction(gameCopy)
		gameCopy.PlayAction(action)
		app.Game.set(gameCopy)
		app.isThinking.set(false)
	}()

	return nil
}

func (app *App) Layout(outsideWidth, outsideHeight int) (int, int) {
	return app.windowWidth(), app.windowHeight()
}

func InitializeApp() *App {
	var blackAgent agents.Agent = agents.NewMCTSAgent(12000, 8, -0.9)
	var whiteAgent agents.Agent = agents.NewMCTSAgent(8000, 8, -0.9)
	var game *environment.Game = environment.NewGame(
		9,   // height
		9,   // width
		6.5, // komi
	)

	var margin Margin = NewMargin(30, 30, 30, 30)
	const BoardSize float32 = 400
	const WindowTitle string = "Go Game"
	const HighlightedIntersectionsRadiusScale float32 = 0.1
	const StoneRadiusScale float32 = 0.4
	const DescriptionBarHeight float32 = 50
	const PassSquareSizeScale float32 = 0.8

	var ui_metadata *UIMetadata = NewUI(WindowTitle, margin, BoardSize, HighlightedIntersectionsRadiusScale, StoneRadiusScale, DescriptionBarHeight, PassSquareSizeScale)
	var app *App = NewApp(blackAgent, whiteAgent, game, ui_metadata)

	ebiten.SetWindowSize(app.windowWidth(), app.windowHeight())
	ebiten.SetWindowTitle(WindowTitle)
	return app
}
