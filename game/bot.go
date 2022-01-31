package game

import (
	"context"
	"time"

	"github.com/notnil/chess"
	"github.com/notnil/chess/uci"
)

type Bot struct {
	ng *uci.Engine
}

func NewBot() (*Bot, error) {
	bot := Bot{}

	ng, err := uci.New("stockfish")
	if err != nil {
		return nil, err
	}
	bot.ng = ng

	return &bot, nil
}

func (b *Bot) Play(fen string, dur time.Duration) (*chess.Move, error) {
	game, err := createGame(fen)
	if err != nil {
		return nil, err
	}

	cmdPos := uci.CmdPosition{Position: game.Position()}
	cmdGo := uci.CmdGo{MoveTime: dur}

	if err := b.ng.Run(uci.CmdUCINewGame, cmdPos, cmdGo); err != nil {
		return nil, err
	}

	return b.ng.SearchResults().BestMove, nil
}

func (b *Bot) Stop() {
	defer b.ng.Close()
}

type BotActivity struct {
	bot *Bot
}

func NewBotActivity(bot *Bot) *BotActivity {
	return &BotActivity{bot}
}

func (b *BotActivity) Execute(ctx context.Context, fen string) (string, error) {
	move, err := b.bot.Play(fen, time.Millisecond*250)
	if err != nil {
		return "", err
	}

	game, err := createGame(fen)
	if err != nil {
		return "", err
	}

	return ChessNotation.Encode(game.Position(), move), nil
}

func createGame(fen string) (*chess.Game, error) {
	opts := []func(*chess.Game){
		chess.UseNotation(ChessNotation),
	}

	fenOpt, err := chess.FEN(fen)
	if err != nil {
		return nil, err
	}
	opts = append(opts, fenOpt)

	return chess.NewGame(opts...), nil
}
