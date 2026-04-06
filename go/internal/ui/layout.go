package ui

func (app *App) WindowHeight() int {
	// Grid extends from 0 to 8, so gridSize = cellSize * 8 = BoardSize * 8 / 9
	gridSize := app.UIMetadata.BoardSize * 8 / 9
	return int(app.UIMetadata.Margin.Top + gridSize + app.UIMetadata.Margin.Bottom + app.UIMetadata.DescriptionBarHeight)
}

func (app *App) WindowWidth() int {
	// Grid extends from 0 to 8, so gridSize = cellSize * 8 = BoardSize * 8 / 9
	gridSize := app.UIMetadata.BoardSize * 8 / 9
	return int(app.UIMetadata.Margin.Left + gridSize + app.UIMetadata.Margin.Right)
}

func (app *App) Layout(outside_width, outside_height int) (int, int) {
	return app.WindowWidth(), app.WindowHeight()
}
