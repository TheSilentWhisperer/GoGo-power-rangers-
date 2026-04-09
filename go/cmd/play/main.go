package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/gen/proto/remote_trainer"
	"github.com/TheSilentWhisperer/GoGo-power-rangers-/go/internal/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	evaluationMode := flag.Bool("eval", false, "Run in evaluation mode: 2000 simulations per move, argmax, no noise")
	tournamentMode := flag.Bool("tournament", false, "Run tournament between 500/1000/2000 simulation agents")
	flag.Parse()

	// Handle tournament mode
	if *tournamentMode {
		conn, err := grpc.NewClient("unix:///tmp/position_evaluator.sock", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			fmt.Println("Error connecting to position evaluation server:", err.Error())
			return
		}
		defer conn.Close()

		var inference_client remote_trainer.PositionEvaluatorClient = remote_trainer.NewPositionEvaluatorClient(conn)
		tm := ui.NewTournamentManager(inference_client)
		tm.RunTournament()
		return
	}

	var app *ui.App = ui.InitializeApp(*evaluationMode)
	if err := ebiten.RunGame(app); err != nil {
		log.Fatal(err)
	}
}
