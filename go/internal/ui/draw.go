package ui

import (
	"image/color"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

func (app *App) CellSize(gameState *GameState) float32 {
	return app.UIMetadata.BoardSize / float32(gameState.Game.Get().Board.Width)
}

func (app *App) HighlightedIntersections(gameState *GameState) []environment.Position {
	var highlighted_positions []environment.Position = make([]environment.Position, 0, 9)
	switch gameState.Game.Get().Board.Height {
	case 9:
		highlighted_positions = []environment.Position{
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
		panic("HighlightedIntersections: unsupported board size")
	}
	return highlighted_positions
}

func (app *App) DrawBackground(ebiten_image *ebiten.Image) {
	// Background with Go board color (dark wood)
	ebiten_image.Fill(color.RGBA{165, 125, 75, 255})
}

func (app *App) DrawGrid(ebiten_image *ebiten.Image, gameState *GameState) {
	// Draw board background with gradient effect
	var offsetX float32 = app.UIMetadata.Margin.Left
	var offsetY float32 = app.UIMetadata.Margin.Top
	boardColor := color.RGBA{195, 150, 100, 255}
	var cellSize float32 = app.CellSize(gameState)
	var gridSize float32 = cellSize * float32(gameState.Game.Get().Board.Width-1)

	vector.FillRect(ebiten_image, offsetX, offsetY, gridSize, gridSize, boardColor, true)

	const stroke_width float32 = 1.5
	var line_color color.Color = color.RGBA{40, 40, 40, 255} // Dark lines for better contrast
	const antialias bool = true

	// Draw horizontal lines (including edges)
	for i := 0; i < gameState.Game.Get().Board.Height; i++ {
		var x0, y0, x1, y1 float32 = offsetX, offsetY + cellSize*float32(i), offsetX + cellSize*float32(gameState.Game.Get().Board.Width-1), offsetY + cellSize*float32(i)
		vector.StrokeLine(ebiten_image, x0, y0, x1, y1, stroke_width, line_color, antialias)
	}

	// Draw vertical lines (including edges)
	for j := 0; j < gameState.Game.Get().Board.Width; j++ {
		var x0, y0, x1, y1 float32 = offsetX + cellSize*float32(j), offsetY, offsetX + cellSize*float32(j), offsetY + cellSize*float32(gameState.Game.Get().Board.Height-1)
		vector.StrokeLine(ebiten_image, x0, y0, x1, y1, stroke_width, line_color, antialias)
	}

	// Draw highlighted intersections (hoshi points) - larger and more visible
	var highlighted_positions []environment.Position = app.HighlightedIntersections(gameState)
	hosiColor := color.RGBA{20, 20, 20, 255}
	for _, pos := range highlighted_positions {
		var cx, cy float32 = offsetX + cellSize*float32(pos.Second), offsetY + cellSize*float32(pos.First)
		var radius float32 = cellSize * 0.08 // Slightly larger hoshi points
		vector.FillCircle(ebiten_image, cx, cy, radius, hosiColor, antialias)
	}
}

func (app *App) DrawStones(ebiten_image *ebiten.Image, gameState *GameState) {
	const antialias bool = true
	var offsetX float32 = app.UIMetadata.Margin.Left
	var offsetY float32 = app.UIMetadata.Margin.Top
	var cellSize float32 = app.CellSize(gameState)

	for i := 0; i < gameState.Game.Get().Board.Height; i++ {
		for j := 0; j < gameState.Game.Get().Board.Width; j++ {
			var stone environment.Stone = gameState.Game.Get().Board.Matrix.Front()[i][j]
			if stone == environment.Empty {
				continue
			}
			var cx, cy float32 = offsetX + cellSize*float32(j), offsetY + cellSize*float32(i)
			var radius float32 = cellSize * app.UIMetadata.StoneRadiusScale

			// Draw shadow for depth
			shadowRadius := radius * 1.05
			shadowColor := color.RGBA{0, 0, 0, 30}
			vector.FillCircle(ebiten_image, cx+1, cy+2, shadowRadius, shadowColor, antialias)

			// Draw main stone with color
			var fill_color color.Color
			switch stone {
			case environment.Black:
				fill_color = color.RGBA{20, 20, 20, 255} // Darker black for better look
				// Draw highlight on top-left for 3D effect
				highlightRadius := radius * 0.4
				highlightColor := color.RGBA{60, 60, 60, 100}
				vector.FillCircle(ebiten_image, cx-radius*0.3, cy-radius*0.3, highlightRadius, highlightColor, antialias)
			case environment.White:
				fill_color = color.RGBA{240, 240, 240, 255} // Slightly off-white for warmth
				// Draw shadow on bottom-right for 3D effect
				shadowHighlightRadius := radius * 0.3
				shadowHighlight := color.RGBA{100, 100, 100, 80}
				vector.FillCircle(ebiten_image, cx+radius*0.35, cy+radius*0.35, shadowHighlightRadius, shadowHighlight, antialias)
			}
			vector.FillCircle(ebiten_image, cx, cy, radius, fill_color, antialias)

			// Draw subtle border for definition
			borderColor := color.RGBA{0, 0, 0, 100}
			vector.StrokeCircle(ebiten_image, cx, cy, radius, 0.8, borderColor, antialias)
		}
	}
}

func (app *App) DescriptionBarWidth() float32 {
	gridSize := app.UIMetadata.BoardSize * 8 / 9
	return app.UIMetadata.Margin.Left + gridSize + app.UIMetadata.Margin.Right - 2*app.UIMetadata.DescriptionBarHeight // We need to leave space for the pass squares
}

func (app *App) DescriptionBarCenter() (float64, float64) {
	gridSize := app.UIMetadata.BoardSize * 8 / 9
	return float64(app.DescriptionBarWidth()) / 2, float64(app.UIMetadata.Margin.Top) + float64(gridSize) + float64(app.UIMetadata.Margin.Bottom) + float64(app.UIMetadata.DescriptionBarHeight)/2
}

func (app *App) DrawDescriptionBar(ebiten_image *ebiten.Image, gameState *GameState) {

	// Write in the bottom margin
	const antialias bool = true
	var font_face font.Face = basicfont.Face7x13
	var text_face text.Face = text.NewGoXFace(font_face)
	var draw_options *text.DrawOptions = &text.DrawOptions{}
	var offsetX float32 = app.UIMetadata.Margin.Left
	var gridSize float32 = app.UIMetadata.BoardSize * 8 / 9
	var offsetY float32 = app.UIMetadata.Margin.Top + gridSize + app.UIMetadata.Margin.Bottom
	var centerX float64 = float64(offsetX + gridSize/2)
	var centerY float64 = float64(offsetY + app.UIMetadata.DescriptionBarHeight/2)
	draw_options.GeoM.Translate(centerX, centerY)
	draw_options.PrimaryAlign = text.AlignCenter
	draw_options.SecondaryAlign = text.AlignCenter
	draw_options.ColorScale.ScaleWithColor(color.RGBA{30, 30, 30, 255}) // Darker text

	// Draw description text
	var description_text string
	if gameState.Game.Get().IsTerminal() {
		var winner environment.Stone = gameState.Game.Get().GetWinner()
		switch winner {
		case environment.Empty:
			description_text = "Game Over: Draw"
		case environment.Black:
			description_text = "Game Over: Black wins"
		case environment.White:
			description_text = "Game Over: White wins"
		}
	} else {
		switch gameState.Game.Get().Board.CurrentPlayer {
		case environment.Black:
			description_text = "Black to play"
		case environment.White:
			description_text = "White to play"
		}
		if gameState.IsThinking.Get() {
			description_text += " (thinking...)"
		}
	}
	text.Draw(ebiten_image, description_text, text_face, draw_options)
}

func (app *App) PassSquareSize() float32 {
	return app.UIMetadata.PassSquareSizeScale * app.UIMetadata.DescriptionBarHeight
}

func (app *App) PassSquareMargin() float32 {
	return (1 - app.UIMetadata.PassSquareSizeScale) / 2 * app.UIMetadata.DescriptionBarHeight
}

func (app *App) PassSquarePosition(gameState *GameState, player environment.Stone) (float32, float32) {
	var offsetX float32 = app.UIMetadata.Margin.Left
	var gridSize float32 = app.UIMetadata.BoardSize * 8 / 9
	var offsetY float32 = app.UIMetadata.Margin.Top + gridSize + app.UIMetadata.Margin.Bottom
	var size float32 = app.PassSquareSize()
	var margin float32 = app.PassSquareMargin()

	switch player {
	case environment.Black:
		return offsetX + gridSize - (3*margin + 2*size), offsetY + margin
	case environment.White:
		return offsetX + gridSize - (margin + size), offsetY + margin
	default:
		panic("PassSquarePosition: invalid player")
	}
}

func (app *App) DrawPassSquare(ebiten_image *ebiten.Image, gameState *GameState) {
	var square_size float32 = app.PassSquareSize()
	var gridSize float32 = app.UIMetadata.BoardSize * 8 / 9
	var offsetY float32 = app.UIMetadata.Margin.Top + gridSize + app.UIMetadata.Margin.Bottom
	var offsetX float32 = app.UIMetadata.Margin.Left

	// Compact positioning - side by side in the right corner
	var spacing float32 = square_size + 4
	var baseX float32 = offsetX + gridSize - (2*spacing + 4)
	var centerY float32 = offsetY + app.UIMetadata.DescriptionBarHeight/2

	// Draw small circular indicators instead of large squares
	var radius float32 = square_size / 2

	// Black pass indicator
	var blackX float32 = baseX
	if gameState.Game.Get().Board.Passes.Black {
		vector.FillCircle(ebiten_image, blackX, centerY, radius, color.RGBA{80, 180, 80, 200}, true)
	} else {
		vector.FillCircle(ebiten_image, blackX, centerY, radius, color.RGBA{200, 200, 200, 150}, true)
	}
	vector.StrokeCircle(ebiten_image, blackX, centerY, radius, 1.5, color.RGBA{40, 40, 40, 200}, true)

	// White pass indicator
	var whiteX float32 = baseX + spacing
	if gameState.Game.Get().Board.Passes.White {
		vector.FillCircle(ebiten_image, whiteX, centerY, radius, color.RGBA{80, 180, 80, 200}, true)
	} else {
		vector.FillCircle(ebiten_image, whiteX, centerY, radius, color.RGBA{200, 200, 200, 150}, true)
	}
	vector.StrokeCircle(ebiten_image, whiteX, centerY, radius, 1.5, color.RGBA{40, 40, 40, 200}, true)
}

func (app *App) Draw(ebiten_image *ebiten.Image) {
	app.DrawBackground(ebiten_image)

	gameState := app.GameState
	var offsetX float32 = app.UIMetadata.Margin.Left
	var offsetY float32 = app.UIMetadata.Margin.Top
	var cellSize float32 = app.CellSize(gameState)

	// Border to frame the board
	var borderPadding float32 = cellSize * 0.6
	var gridSize float32 = cellSize * float32(gameState.Game.Get().Board.Width-1)
	borderColor := color.RGBA{60, 40, 20, 255}
	vector.StrokeRect(ebiten_image, offsetX-borderPadding, offsetY-borderPadding, gridSize+2*borderPadding, gridSize+2*borderPadding, 3, borderColor, true)

	app.DrawGrid(ebiten_image, gameState)
	app.DrawStones(ebiten_image, gameState)
	app.DrawDescriptionBar(ebiten_image, gameState)
	app.DrawPassSquare(ebiten_image, gameState)
}
