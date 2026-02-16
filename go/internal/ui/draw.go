package ui

import (
	"image/color"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

func (app *App) CellSize() float32 {
	return app.UIMetadata.BoardSize / float32(app.Game.Get().Board.Width-1)
}

func (app *App) HighlightedIntersections() []environment.Position {
	var highlighted_positions []environment.Position = make([]environment.Position, 0, 9)
	switch app.Game.Get().Board.Height {
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
	// Background
	ebiten_image.Fill(color.RGBA{200, 170, 120, 255})
}

func (app *App) DrawGrid(ebiten_image *ebiten.Image) {

	const stroke_width float32 = 2
	var line_color color.Color = color.Black
	const antialias bool = true

	// Draw horizontal lines
	for i := 0; i < app.Game.Get().Board.Height; i++ {
		var x0, y0, x1, y1 float32 = app.UIMetadata.Margin.Left, app.UIMetadata.Margin.Top + app.CellSize()*float32(i), app.UIMetadata.BoardSize + app.UIMetadata.Margin.Left, app.UIMetadata.Margin.Top + app.CellSize()*float32(i)
		vector.StrokeLine(ebiten_image, x0, y0, x1, y1, stroke_width, line_color, antialias)
	}

	// Draw vertical lines
	for j := 0; j < app.Game.Get().Board.Width; j++ {
		var x0, y0, x1, y1 float32 = app.UIMetadata.Margin.Left + app.CellSize()*float32(j), app.UIMetadata.Margin.Top, app.UIMetadata.Margin.Left + app.CellSize()*float32(j), app.UIMetadata.BoardSize + app.UIMetadata.Margin.Top
		vector.StrokeLine(ebiten_image, x0, y0, x1, y1, stroke_width, line_color, antialias)
	}

	// Draw highlighted intersections
	var highlighted_positions []environment.Position = app.HighlightedIntersections()
	for _, pos := range highlighted_positions {
		var cx, cy float32 = app.UIMetadata.Margin.Left + app.CellSize()*float32(pos.Second), app.UIMetadata.Margin.Top + app.CellSize()*float32(pos.First)
		var radius float32 = app.CellSize() * app.UIMetadata.HighlightedIntersectionsRadiusScale
		vector.FillCircle(ebiten_image, cx, cy, radius, line_color, antialias)
	}
}

func (app *App) DrawStones(ebiten_image *ebiten.Image) {
	const antialias bool = true
	for i := 0; i < app.Game.Get().Board.Height; i++ {
		for j := 0; j < app.Game.Get().Board.Width; j++ {
			var stone environment.Stone = app.Game.Get().Board.Matrix[i][j]
			if stone == environment.Empty {
				continue
			}
			var cx, cy float32 = app.UIMetadata.Margin.Left + app.CellSize()*float32(j), app.UIMetadata.Margin.Top + app.CellSize()*float32(i)
			var radius float32 = app.CellSize() * app.UIMetadata.StoneRadiusScale
			var fill_color color.Color
			switch stone {
			case environment.Black:
				fill_color = color.Black
			case environment.White:
				fill_color = color.White
			}
			vector.FillCircle(ebiten_image, cx, cy, radius, fill_color, antialias)
		}
	}
}

func (app *App) DescriptionBarWidth() float32 {
	return app.UIMetadata.Margin.Left + app.UIMetadata.BoardSize + app.UIMetadata.Margin.Right - 2*app.UIMetadata.DescriptionBarHeight // We need to leave space for the pass squares
}

func (app *App) DescriptionBarCenter() (float64, float64) {
	return float64(app.DescriptionBarWidth()) / 2, float64(app.UIMetadata.Margin.Top) + float64(app.UIMetadata.BoardSize) + float64(app.UIMetadata.Margin.Bottom) + float64(app.UIMetadata.DescriptionBarHeight)/2
}

func (app *App) DrawDescriptionBar(ebiten_image *ebiten.Image) {

	// Write in the bottom margin
	const antialias bool = true
	var font_face font.Face = basicfont.Face7x13
	var text_face text.Face = text.NewGoXFace(font_face)
	var draw_options *text.DrawOptions = &text.DrawOptions{}
	var text_center_x, text_center_y float64 = app.DescriptionBarCenter()
	draw_options.GeoM.Translate(text_center_x, text_center_y)
	draw_options.PrimaryAlign = text.AlignCenter
	draw_options.SecondaryAlign = text.AlignCenter

	// Draw description text
	var description_text string
	if app.Game.Get().IsTerminal() {
		var winner environment.Stone = app.Game.Get().GetWinner()
		switch winner {
		case environment.Empty:
			description_text = "Game over: Draw"
		case environment.Black:
			description_text = "Game over: Black wins"
		case environment.White:
			description_text = "Game over: White wins"
		}
	} else {
		switch app.Game.Get().Board.CurrentPlayer {
		case environment.Black:
			description_text = "Black"
		case environment.White:
			description_text = "White"
		}
		switch app.IsThinking.Get() {
		case true:
			description_text += " is thinking..."
		default:
			description_text += " is ready to play!"
		}
	}
	text.Draw(ebiten_image, description_text, text_face, draw_options)
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

func (app *App) DrawPassSquare(ebiten_image *ebiten.Image) {

	var square_size float32 = app.PassSquareSize()
	// background colors for the pass squares
	var not_passed_background_color color.Color = color.RGBA{255, 0, 0, 50}
	var passed_background_color color.Color = color.RGBA{0, 255, 0, 50}

	//draw square for black
	var x, y float32 = app.PassSquarePosition(environment.Black)
	if app.Game.Get().Board.Passes.Black {
		vector.FillRect(ebiten_image, x, y, square_size, square_size, passed_background_color, true)
	} else {
		vector.FillRect(ebiten_image, x, y, square_size, square_size, not_passed_background_color, true)
	}
	vector.StrokeRect(ebiten_image, x, y, square_size, square_size, 2, color.Black, true)

	//draw square for white
	x, y = app.PassSquarePosition(environment.White)
	if app.Game.Get().Board.Passes.White {
		vector.FillRect(ebiten_image, x, y, square_size, square_size, passed_background_color, true)
	} else {
		vector.FillRect(ebiten_image, x, y, square_size, square_size, not_passed_background_color, true)
	}
	vector.StrokeRect(ebiten_image, x, y, square_size, square_size, 2, color.White, true)
}

func (app *App) Draw(ebiten_image *ebiten.Image) {
	app.DrawBackground(ebiten_image)
	app.DrawGrid(ebiten_image)
	app.DrawStones(ebiten_image)
	app.DrawDescriptionBar(ebiten_image)
	app.DrawPassSquare(ebiten_image)
}
