package ui

import (
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/agents"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/environment"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/utils"
	"github.com/hajimehoshi/ebiten/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

type GameState struct {
	Game            *utils.LockedPointer[environment.Game]
	BlackAgent      agents.Agent
	WhiteAgent      agents.Agent
	IsThinking      *utils.LockedBool
	PositionHistory []*utils.LockedPair[*environment.Game, []int]
}

type App struct {
	MoveSearchInitiated chan bool // Channel to signal the start of move search (used for synchronization between the main thread and the MCTS goroutine)
	IsPaused            *utils.LockedBool
	GameState           *GameState
	UIMetadata          *UIMetadata
	KeyStates           map[ebiten.Key]*utils.LockedPointer[KeyState]
	Client              remote_trainer.NetTrainerClient
	EvaluationMode      bool // If true, skip training data submission
}

func NewApp(game *environment.Game, ui_metadata *UIMetadata, key_list []ebiten.Key, inference_client remote_trainer.PositionEvaluatorClient, training_client remote_trainer.NetTrainerClient, evaluation_mode bool) *App {
	var app *App = &App{
		MoveSearchInitiated: make(chan bool, 1),
		IsPaused:            utils.NewLockedBool(false),
		UIMetadata:          ui_metadata,
		KeyStates:           make(map[ebiten.Key]*utils.LockedPointer[KeyState]),
		Client:              training_client,
		EvaluationMode:      evaluation_mode,
	}

	// Create agents based on mode
	var black_agent agents.Agent
	var white_agent agents.Agent

	if evaluation_mode {
		// Evaluation mode: 2000 simulations, argmax, no noise
		black_agent = agents.NewPuctAgentEval(5000, 8, 16, -0.9, inference_client)
		white_agent = agents.NewPuctAgentEval(5000, 8, 16, -0.9, inference_client)
	} else {
		// Training mode: 400 simulations, temperature sampling, Dirichlet noise
		black_agent = agents.NewPuctAgent(400, 8, 16, -0.99, inference_client, 0.25, 10)
		white_agent = agents.NewPuctAgent(400, 8, 16, -0.99, inference_client, 0.25, 10)
	}

	app.GameState = &GameState{
		Game:            utils.NewLockedPointer(game),
		BlackAgent:      black_agent,
		WhiteAgent:      white_agent,
		IsThinking:      utils.NewLockedBool(false),
		PositionHistory: make([]*utils.LockedPair[*environment.Game, []int], 0),
	}

	for _, key := range key_list {
		app.KeyStates[key] = utils.NewLockedPointer[KeyState](NewKeyState(key))
	}
	return app
}

func InitializeApp(evaluation_mode bool) *App {
	//establish UDS connection to the position evaluation server
	conn, err := grpc.NewClient("unix:///tmp/position_evaluator.sock", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		println("Error connecting to position evaluation server:", err.Error())
		return nil
	}

	var inference_client remote_trainer.PositionEvaluatorClient = remote_trainer.NewPositionEvaluatorClient(conn)
	var training_client remote_trainer.NetTrainerClient = remote_trainer.NewNetTrainerClient(conn)

	// Create single game
	var game *environment.Game = environment.NewGame(
		9,   // height
		9,   // width
		4,   // history length
		6.5, // komi
	)

	var margin Margin = NewMargin(80, 80, 80, 80)
	const BoardSize float32 = 500
	const WindowTitle string = "Go Game"
	const HighlightedIntersectionsRadiusScale float32 = 0.1
	const StoneRadiusScale float32 = 0.4
	const DescriptionBarHeight float32 = 40
	const PassSquareSizeScale float32 = 0.4
	var KeyList []ebiten.Key = []ebiten.Key{ebiten.KeySpace}

	var ui_metadata *UIMetadata = NewUI(WindowTitle, margin, BoardSize, HighlightedIntersectionsRadiusScale, StoneRadiusScale, DescriptionBarHeight, PassSquareSizeScale)
	var app *App = NewApp(game, ui_metadata, KeyList, inference_client, training_client, evaluation_mode)

	ebiten.SetWindowSize(app.WindowWidth(), app.WindowHeight())
	ebiten.SetWindowTitle(WindowTitle)

	// Center window on primary monitor
	monitor := ebiten.Monitor()
	monitorW, monitorH := monitor.Size()
	windowW := app.WindowWidth()
	windowH := app.WindowHeight()
	centerX := (monitorW - windowW) / 2
	centerY := (monitorH - windowH) / 2
	ebiten.SetWindowPosition(centerX, centerY)

	return app
}
