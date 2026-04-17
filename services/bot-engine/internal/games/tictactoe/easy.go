package tictactoe

import "math/rand"

type Move struct {
	Cell string
}

var winningLines = [][3]int{
	{0, 1, 2},
	{3, 4, 5},
	{6, 7, 8},
	{0, 3, 6},
	{1, 4, 7},
	{2, 5, 8},
	{0, 4, 8},
	{2, 4, 6},
}

var corners = []int{0, 2, 6, 8}
var sides = []int{1, 3, 5, 7}
var indexToCell = []string{
	"a1", "b1", "c1",
	"a2", "b2", "c2",
	"a3", "b3", "c3",
}

func SelectEasyMove(board string, turn string) (*Move, error) {
	state, err := normalizeBoard(board)
	if err != nil {
		return nil, err
	}

	botMark := byte('x')
	opponentMark := byte('o')
	if turn == "b" {
		botMark = 'o'
		opponentMark = 'x'
	}

	available := availableMoves(state)
	if len(available) == 0 {
		return nil, nil
	}

	if cell, ok := firstWinningMove(state, botMark, available); ok {
		return &Move{Cell: indexToCell[cell]}, nil
	}

	if cell, ok := firstWinningMove(state, opponentMark, available); ok {
		return &Move{Cell: indexToCell[cell]}, nil
	}

	if state[4] == '-' {
		return &Move{Cell: indexToCell[4]}, nil
	}

	if cell, ok := pickPreferredMove(state, corners); ok {
		return &Move{Cell: indexToCell[cell]}, nil
	}

	if cell, ok := pickPreferredMove(state, sides); ok {
		return &Move{Cell: indexToCell[cell]}, nil
	}

	choice := available[rand.Intn(len(available))]
	return &Move{Cell: indexToCell[choice]}, nil
}

func normalizeBoard(board string) ([]byte, error) {
	if len(board) != len(indexToCell) {
		return nil, ErrInvalidBoard
	}

	state := []byte(board)
	for _, value := range state {
		if value != '-' && value != 'x' && value != 'o' {
			return nil, ErrInvalidBoard
		}
	}

	return state, nil
}

func availableMoves(state []byte) []int {
	moves := make([]int, 0, len(state))
	for index, value := range state {
		if value == '-' {
			moves = append(moves, index)
		}
	}
	return moves
}

func firstWinningMove(state []byte, mark byte, candidates []int) (int, bool) {
	for _, candidate := range candidates {
		if completesLine(state, candidate, mark) {
			return candidate, true
		}
	}
	return 0, false
}

func pickPreferredMove(state []byte, candidates []int) (int, bool) {
	available := make([]int, 0, len(candidates))
	for _, candidate := range candidates {
		if state[candidate] == '-' {
			available = append(available, candidate)
		}
	}
	if len(available) == 0 {
		return 0, false
	}
	return available[rand.Intn(len(available))], true
}

func completesLine(state []byte, candidate int, mark byte) bool {
	for _, line := range winningLines {
		inLine := false
		for _, index := range line {
			if index == candidate {
				inLine = true
				break
			}
		}
		if !inLine {
			continue
		}

		win := true
		for _, index := range line {
			value := state[index]
			if index == candidate {
				value = mark
			}
			if value != mark {
				win = false
				break
			}
		}
		if win {
			return true
		}
	}

	return false
}

type invalidBoardError struct{}

func (invalidBoardError) Error() string { return "invalid tic-tac-toe board" }

var ErrInvalidBoard error = invalidBoardError{}
