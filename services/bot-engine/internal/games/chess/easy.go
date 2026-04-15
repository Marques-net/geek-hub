package chess

import (
	"math/rand"

	"github.com/notnil/chess"
)

type Move struct {
	From      string
	To        string
	Promotion string
}

type weightedMove struct {
	move   *chess.Move
	weight int
}

var centerSquares = map[string]struct{}{
	"d4": {},
	"e4": {},
	"d5": {},
	"e5": {},
}

func SelectEasyMove(fen string) (*Move, error) {
	fenOption, err := chess.FEN(fen)
	if err != nil {
		return nil, err
	}

	game := chess.NewGame(fenOption)
	validMoves := game.ValidMoves()
	if len(validMoves) == 0 {
		return nil, nil
	}

	weightedMoves := make([]weightedMove, 0, len(validMoves))
	totalWeight := 0

	for _, move := range validMoves {
		weight := 1

		if move.HasTag(chess.Capture) {
			weight += 4
		}

		if move.Promo() != chess.NoPieceType {
			weight += 4
		}

		if move.HasTag(chess.Check) {
			weight += 2
		}

		if _, ok := centerSquares[move.S2().String()]; ok {
			weight += 1
		}

		weightedMoves = append(weightedMoves, weightedMove{
			move:   move,
			weight: weight,
		})
		totalWeight += weight
	}

	roll := rand.Intn(totalWeight)
	selected := weightedMoves[len(weightedMoves)-1].move

	for _, candidate := range weightedMoves {
		if roll < candidate.weight {
			selected = candidate.move
			break
		}

		roll -= candidate.weight
	}

	return &Move{
		From:      selected.S1().String(),
		To:        selected.S2().String(),
		Promotion: promotionToString(selected.Promo()),
	}, nil
}

func promotionToString(pieceType chess.PieceType) string {
	switch pieceType {
	case chess.Queen:
		return "q"
	case chess.Rook:
		return "r"
	case chess.Bishop:
		return "b"
	case chess.Knight:
		return "n"
	default:
		return ""
	}
}
