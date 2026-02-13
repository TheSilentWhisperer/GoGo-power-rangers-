package ui

func (app *App) WindowHeight() int {
	return int(app.UIMetadata.Margin.Top + app.UIMetadata.BoardSize + app.UIMetadata.Margin.Bottom + app.UIMetadata.DescriptionBarHeight)
}

func (app *App) WindowWidth() int {
	return int(app.UIMetadata.Margin.Left + app.UIMetadata.BoardSize + app.UIMetadata.Margin.Right)
}

func (app *App) Layout(outside_width, outside_height int) (int, int) {
	return app.WindowWidth(), app.WindowHeight()
}
