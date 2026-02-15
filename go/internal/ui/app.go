package ui

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/gen/proto/position_evaluation"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/internal/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"google.golang.org/grpc"
)

// Locked types (LockedBool, LockedGame, LockedValue) moved to internal/utils.

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

func NewUI(window_title string, margin Margin, board_size, highlighted_intersections_radius_scale, stone_radius_scale, description_bar_height, pass_square_size_scale float32) *UIMetadata {
	return &UIMetadata{
		WindowTitle:                         window_title,
		Margin:                              margin,
		BoardSize:                           board_size,
		HighlightedIntersectionsRadiusScale: highlighted_intersections_radius_scale,
		StoneRadiusScale:                    stone_radius_scale,
		DescriptionBarHeight:                description_bar_height,
		PassSquareSizeScale:                 pass_square_size_scale,
	}
}

type App struct {
	MoveSearchInitiated chan bool // Channel to signal the start of move search (used for synchronization between the main thread and the MCTS goroutine)
	IsThinking          *utils.LockedBool
	IsPaused            *utils.LockedBool
	BlackAgent          agents.Agent
	WhiteAgent          agents.Agent
	Game                *utils.LockedPointer[environment.Game]
	UIMetadata          *UIMetadata
	KeyStates           map[ebiten.Key]*utils.LockedPointer[KeyState]
}

func NewApp(black_agent, white_agent agents.Agent, game *environment.Game, ui_metadata *UIMetadata, key_list []ebiten.Key) *App {
	var app *App = &App{
		MoveSearchInitiated: make(chan bool, 1),
		IsThinking:          utils.NewLockedBool(false), // Whether the current agent is thinking
		IsPaused:            utils.NewLockedBool(false), // Whether the game is paused
		BlackAgent:          black_agent,
		WhiteAgent:          white_agent,
		Game:                utils.NewLockedPointer(game),
		UIMetadata:          ui_metadata,
		KeyStates:           make(map[ebiten.Key]*utils.LockedPointer[KeyState]),
	}
	for _, key := range key_list {
		app.KeyStates[key] = utils.NewLockedPointer[KeyState](NewKeyState(key))
	}
	return app
}

func InitializeApp() *App {

	//establish UDS connection to the position evaluation server
	conn, err := grpc.NewClient("unix:///tmp/position_evaluation.sock")
	if err != nil {
		println("Error connecting to position evaluation server:", err.Error())
		return nil
	}

	client := position_evaluation.NewPositionEvaluatorClient(conn)

	var black_agent agents.Agent = agents.NewPUCTAgent(5000, 8, -0.7, client)
	var white_agent agents.Agent = agents.NewMCTSAgent((000, 8, -0.7)
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
	var KeyList []ebiten.Key = []ebiten.Key{ebiten.KeySpace}

	var ui_metadata *UIMetadata = NewUI(WindowTitle, margin, BoardSize, HighlightedIntersectionsRadiusScale, StoneRadiusScale, DescriptionBarHeight, PassSquareSizeScale)
	var app *App = NewApp(black_agent, white_agent, game, ui_metadata, KeyList)

	ebiten.SetWindowSize(app.WindowWidth(), app.WindowHeight())
	ebiten.SetWindowTitle(WindowTitle)
	return app
}
