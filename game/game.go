package game

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	"github.com/notnil/chess"
	"go.temporal.io/sdk/workflow"
)

var ChessNotation = chess.UCINotation{}

type GameWorkflowParams struct {
	Color Color  `json:"color"` // Color chosen by the user, leave empty for a random pick.
	FEN   string `json:"fen"`   // Initial state of the game in Forsysth-Edwards notation.
}

func (params *GameWorkflowParams) PickColor(ctx workflow.Context) {
	if params.Color != NoColor {
		return
	}

	var pick Color
	workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
		colors := []chess.Color{chess.White, chess.Black}
		src := rand.NewSource(time.Now().UnixNano())
		rnd := rand.New(src)

		pick := Color(colors[rnd.Intn(len(colors))])
		return pick
	}).Get(&pick)

	params.Color = pick
}

// GameInfo is the data structure that this workflow is going to return from
// query handlers or as the final return value describing the state of the game
type GameInfo struct {
	FEN        string        // Position of the board.
	Outcome    chess.Outcome // Result of the game ("*" means "in progress").
	Method     chess.Method  // Method that generated the outcome.
	Board      string        // Simple viz of the board using Unicode chess symbols.
	Turn       Turn          // Current turn.
	Color      Color         // User's color.
	ValidMoves []string      // Valid moves when it is the user's turn.
}

// NewInfoFromGame creates a new GameInfo.
func NewInfoFromGame(g *chess.Game, c Color, t Turn) *GameInfo {
	info := GameInfo{
		FEN:     g.FEN(),
		Outcome: g.Outcome(),
		Method:  g.Method(),
		Board:   g.Position().Board().Draw(),
		Turn:    t,
		Color:   c,
	}

	if t == User {
		info.ValidMoves = validMoves(g)
	}

	return &info
}

// Color is a wrap of chess.Color with JSON-encoding compatibility.
type Color chess.Color

const (
	NoColor Color = Color(chess.NoColor)
	White   Color = Color(chess.White)
	Black   Color = Color(chess.Black)
)

func (c Color) String() string {
	return c.Chess().Name()
}

func (c Color) Chess() chess.Color {
	return chess.Color(c)
}

func (c Color) MarshalJSON() ([]byte, error) {
	return json.Marshal(chess.Color(c).Name())
}

func (c *Color) UnmarshalJSON(blob []byte) error {
	var str string
	if err := json.Unmarshal(blob, &str); err != nil {
		return err
	}

	*c = NoColor

	switch strings.ToLower(str) {
	case "white", "w":
		*c = White
	case "black", "b":
		*c = Black
	}

	return nil
}

// Turn of the game.
type Turn uint8

const (
	User    Turn = 0
	Machine Turn = 1
)

func (t *Turn) Shift() {
	*t = (Turn)((uint8)(*t) ^ 1)
}

func (t Turn) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func (t *Turn) UnmarshalJSON(blob []byte) error {
	var str string
	if err := json.Unmarshal(blob, &str); err != nil {
		return err
	}

	var turn Turn
	switch string(str) {
	case "User":
		turn = User
	case "Machine":
		turn = Machine
	}

	*t = turn

	return nil
}

func (t Turn) String() string {
	switch t {
	case User:
		return "User"
	case Machine:
		return "Machine"
	}
	return ""
}

// GameWorkflow is the game workflow function.
func GameWorkflow(ctx workflow.Context, params GameWorkflowParams) (*GameInfo, error) {
	logger := workflow.GetLogger(ctx)

	params.PickColor(ctx)
	logger.Info("New game", "user", params.Color)

	// Options of the game.
	opts := []func(*chess.Game){
		chess.UseNotation(ChessNotation),
	}
	if params.FEN != "" {
		fen, err := chess.FEN(params.FEN)
		if err != nil {
			return nil, err
		}
		opts = append(opts, fen)
	}

	// Create new game.
	game := chess.NewGame(opts...)

	// Decide who goes first.
	turn := User
	if params.Color == Black {
		turn = Machine
	}

	// Query handler to provide the state of the game.
	workflow.SetQueryHandler(ctx, "info", func() (*GameInfo, error) {
		return NewInfoFromGame(game, params.Color, turn), nil
	})

	// Couple of signals so the user can decide the next move.
	resignSignalChan := workflow.GetSignalChannel(ctx, "resign")
	moveSignalChan := workflow.GetSignalChannel(ctx, "move")
	moveRequest := ""

	// Create selector to consume the signal channels.
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(resignSignalChan, func(ch workflow.ReceiveChannel, _ bool) {
		var signal interface{}
		ch.Receive(ctx, &signal)

		game.Resign(params.Color.Chess())
	})
	selector.AddReceive(moveSignalChan, func(ch workflow.ReceiveChannel, _ bool) {
		ch.Receive(ctx, &moveRequest)
	})

	// Game loop.
	for game.Outcome() == chess.NoOutcome {
		var err error

		// Machine's turn.
		if turn == Machine {
			// Pick the next move randomly. An interesting exercise for the
			// student could be to refactor this piece using workflow activities
			// and have an activity worker play the game using a chess engine
			// like stockfish.
			if err = machinesMove(ctx, game, turn); err != nil {
				return NewInfoFromGame(game, params.Color, turn), err
			}
		}

		// User's turn.
		if turn == User {
			// Block until the user sends us the next move.
			moveRequest = ""
			selector.Select(ctx)
			if moveRequest != "" {
				err = game.MoveStr(moveRequest)
			}
		}

		if err != nil {
			continue
		}

		turn.Shift()
	}

	logger.Warn("Game over!", "outcome", game.Outcome().String())

	return NewInfoFromGame(game, params.Color, turn), nil
}

// machinesMove is a naive chess player that picks moves randomly.
func machinesMove(ctx workflow.Context, game *chess.Game, turn Turn) error {
	var move string
	opts := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		TaskQueue:              "queue",
		ScheduleToStartTimeout: time.Second * 2,
		StartToCloseTimeout:    time.Second * 2,
	})
	err := workflow.ExecuteActivity(opts, "play", game.FEN()).Get(ctx, &move)
	if err != nil {
		return err
	}

	return game.MoveStr(move)

	/*
		// We use the SideEffect API because it records its output in the workflow
		// history providing the required determinism.
		move := ""
		moves := validMoves(game)
		workflow.SideEffect(ctx, func(ctx workflow.Context) interface{} {
			move := moves[rand.Intn(len(moves))]
			return move
		}).Get(&move)

		return game.MoveStr(move)
	*/
}

// validMoves returns a slice of valid moves encoded using ChessNotation.
func validMoves(game *chess.Game) []string {
	pos := game.Position()
	moves := game.ValidMoves()

	ret := make([]string, len(moves))
	for i, m := range moves {
		ret[i] = ChessNotation.Encode(pos, m)
	}

	return ret
}
