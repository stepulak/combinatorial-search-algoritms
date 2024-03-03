package csa

import (
	"strings"
	"testing"
)

const (
	cross int = iota - 1
	empty
	circle
)

// Basic node struct
// Intentionally passed by value everywhere
type tttNode struct {
	board [3][3]int
}

func (node tttNode) Score() int {
	_, symbol := node.anyFullRow()
	return symbol * (node.numberEmptySquares() + 1)
}

func (node tttNode) IsTerminal() bool {
	row, _ := node.anyFullRow()
	return row || node.numberEmptySquares() == 0
}

func (node tttNode) SearchNodeGenerator() SearchNodeGenerator {
	symbol := map[bool]int{true: circle, false: cross}
	x, y := 0, 0
	return func(maximizing bool) SearchNode {
		for y < 3 {
			for x < 3 {
				if node.board[y][x] == empty {
					nodeCopy := node
					nodeCopy.board[y][x] = symbol[maximizing]
					x++
					return nodeCopy
				}
				x++
			}
			x = 0
			y++
		}
		return nil
	}
}

func (node tttNode) String() string {
	sb := strings.Builder{}
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			t := node.board[y][x]
			if t == empty {
				sb.WriteString("_ ")
			} else if t == cross {
				sb.WriteString("X ")
			} else if t == circle {
				sb.WriteString("O ")
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (node tttNode) numberEmptySquares() int {
	num := 0
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			if node.board[y][x] == empty {
				num++
			}
		}
	}
	return num
}

// Check whether there is any straight full row/column/diagonal with three symbols
func (node tttNode) anyFullRow() (bool, int) {
	b := &node.board
	for i := 0; i < 3; i++ {
		if isInRow(b[i][0], b[i][1], b[i][2]) {
			return true, b[i][0]
		}
		if isInRow(b[0][i], b[1][i], b[2][i]) {
			return true, b[0][i]
		}
	}
	if isInRow(b[0][0], b[1][1], b[2][2]) || isInRow(b[2][0], b[1][1], b[0][2]) {
		return true, b[1][1]
	}
	return false, empty
}

func isInRow(a, b, c int) bool {
	return a != empty && a == b && b == c
}

func TestTTTScoreAndIsTerminal(t *testing.T) {
	node := tttNode{}
	if node.Score() != 0 {
		t.Error("Empty node must have 0 score")
	}
	if node.IsTerminal() {
		t.Error("Empty node cannot be terminal")
	}
	node.board[0][0] = cross
	node.board[0][1] = cross
	if node.Score() != 0 {
		t.Error("Node without row must have 0 score")
	}
	if node.IsTerminal() {
		t.Error("Node without row cannot be terminal")
	}
	node.board[0][2] = circle
	if node.Score() != 0 {
		t.Error("Node without proper row must have 0 score")
	}
	if node.IsTerminal() {
		t.Error("Node without proper row cannot be terminal")
	}
	node.board[0][2] = cross
	if node.Score() > cross {
		t.Error("Node with proper row must have non-zero score")
	}
	if !node.IsTerminal() {
		t.Error("Node with proper row must be terminal")
	}
	node = tttNode{}
	node.board[0][0] = circle
	node.board[1][1] = circle
	node.board[2][2] = circle
	if node.Score() < circle {
		t.Error("Node with proper row must have non-zero score")
	}
	if !node.IsTerminal() {
		t.Error("Node with proper row must be terminal")
	}
}

func TestTTTString(t *testing.T) {
	node := tttNode{}
	if node.String() != "_ _ _ \n_ _ _ \n_ _ _ \n" {
		t.Errorf("Invalid string for empty board: %s", node.String())
	}
	node.board[0][1] = cross
	node.board[1][0] = cross
	node.board[1][1] = circle
	node.board[1][2] = cross
	node.board[2][1] = cross
	if node.String() != "_ X _ \nX O X \n_ X _ \n" {
		t.Errorf("Invalid string for board: %s", node.String())
	}
}

func TestTTTSearchNodeGenerator(t *testing.T) {
	generator := tttNode{}.SearchNodeGenerator()
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			node := generator(false)
			if node == nil {
				t.Error("Generator must return node")
			}
			if node.(tttNode).board[y][x] != cross {
				t.Errorf("Generator returned invalid node %s", node)
			}
		}
	}
	if n := generator(false); n != nil {
		t.Error("Generator must return nil")
	}
}

func TestTTTSingleMinimax(t *testing.T) {
	var sn SearchNode = tttNode{}
	for i := 0; i < 9; i++ {
		newNode, _ := Minimax(sn, 9, false)
		sn = newNode
	}
	if !sn.IsTerminal() {
		t.Errorf("Not finishing on terminal node %s", sn)
	}
	if sn.Score() >= empty {
		t.Errorf("Cross did not win %s", sn)
	}
}

func TestTTTBestMinimaxVsBestMinimax(t *testing.T) {
	var sn SearchNode = tttNode{}
	maximizing := false
	for i := 0; i < 9; i++ {
		newNode, _ := Minimax(sn, 9, maximizing)
		sn = newNode
		maximizing = !maximizing
	}
	if !sn.IsTerminal() {
		t.Errorf("Not finishing on terminal node %s", sn)
	}
	if sn.Score() != 0 {
		t.Errorf("Score is not a draw %s", sn)
	}
}

func TestTTTMinimaxAlphaBetaPrunning(t *testing.T) {
	var sn SearchNode = tttNode{}
	maximizing := true
	for i := 0; i < 9; i++ {
		newNode, _ := MinimaxAlphaBetaPrunning(sn, 9, maximizing)
		sn = newNode
		maximizing = !maximizing
	}
	if !sn.IsTerminal() {
		t.Errorf("Not finishing on terminal node %s", sn)
	}
	if sn.Score() != empty {
		t.Errorf("Score is not a draw %s", sn)
	}
}

func TestTTTBestMinimaxVsWorseMinimax(t *testing.T) {
	type minimax func(node SearchNode, depth int, maximizing bool) (SearchNode, int)

	runTest := func(minimaxFn minimax, depths map[bool]int, moves int) {
		var sn SearchNode = tttNode{}
		maximizing := false
		// has to win in the least number of moves
		for i := 0; i < moves; i++ {
			newNode, _ := minimaxFn(sn, depths[maximizing], maximizing)
			sn = newNode
			maximizing = !maximizing
		}
		if !sn.IsTerminal() {
			t.Errorf("Not finishing on terminal node %s", sn)
		}
		if sn.Score() >= empty {
			t.Errorf("Cross did not win %s", sn)
		}
	}
	runTest(Minimax, map[bool]int{false: 9, true: 1}, 5)
	runTest(Minimax, map[bool]int{false: 9, true: 2}, 7)
	runTest(MinimaxAlphaBetaPrunning, map[bool]int{false: 9, true: 1}, 5)
	runTest(MinimaxAlphaBetaPrunning, map[bool]int{false: 9, true: 2}, 7)

	minimaxConcurrent := func(node SearchNode, depth int, maximizing bool) (SearchNode, int) {
		return MinimaxConcurrent(node, depth, maximizing, 2)
	}
	runTest(minimaxConcurrent, map[bool]int{false: 9, true: 1}, 5)
	runTest(minimaxConcurrent, map[bool]int{false: 9, true: 2}, 7)
}
