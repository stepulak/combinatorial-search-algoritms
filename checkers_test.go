package csa

import (
	"reflect"
	"strings"
	"testing"
)

// Rules of this checkers game:
// - king can move and jump only by one square
// - there is no necessity for a jump if available; if the figure wont jump it wont be taken away

// Algorithm:
// - anticycling technique using node history
// - draw detection if there is no more moves without cycling
// - win/loss detection (all enemy pieces are dead)
// - node is considered terminal if it's win/loss
// - if player can't move, it's not a draw (unlike in chess)

const (
	// single figure scores
	pawnScore = 1
	kingScore = 3

	// color indices
	white = 0
	black = 1

	// figure indices
	pawns = 0
	kings = 1

	// directions on board
	whitePawnDir = -1
	blackPawnDir = 1

	// score coefficient
	whiteCoef = -1
	blackCoef = 1

	// runes
	whitePawn = '♟'
	whiteKing = '♛'
	blackPawn = '♙'
	blackKing = '♕'
)

type cNodeHistory map[uint64][]cNode

// Basic node struct
// Intentionally passed by value everywhere
type cNode struct {
	board       [2][2]uint64 // board[units][color]
	nodeHistory cNodeHistory // always passed by reference
}

func (node cNode) Score() int {
	score := 0
	// There is actually a better option - store score and modify it after each move (add/subtract move's difference)
	// Recalculate everytime for sake of simplicity (relax it's just a test not a professional checkers engine)
	for i := 0; i < 64; i++ {
		if isBit(node.board[pawns][white], i) {
			score += pawnScore * whiteCoef
		} else if isBit(node.board[kings][white], i) {
			score += kingScore * whiteCoef
		} else if isBit(node.board[pawns][black], i) {
			score += pawnScore * blackCoef
		} else if isBit(node.board[kings][black], i) {
			score += kingScore * blackCoef
		}
	}
	return score
}

func (node cNode) IsTerminal() bool {
	for color := range []int{white, black} {
		if node.board[pawns][color]|node.board[kings][color] == 0 {
			// this color is no more => node is terminal
			return true
		}
	}
	return false
}

func (node cNode) SearchNodeGenerator() SearchNodeGenerator {
	var nodeQueue []cNode
	index := 0
	return func(maximizing bool) SearchNode {
		if len(nodeQueue) == 0 {
			// nodeQueue is empty, generate more moves if possible
			// maximizing = black moves
			// minimizing = white moves
			var color, pawnDir int
			if maximizing {
				pawnDir = blackPawnDir
				color = black
			} else {
				pawnDir = whitePawnDir
				color = white
			}
			for ; index < 64; index++ {
				if node.placeOccupiedFigureColor(pawns, color, index) {
					nodeQueue = node.generatePawnMoves(color, index, pawnDir)
				} else if node.placeOccupiedFigureColor(kings, color, index) {
					nodeQueue = node.generateKingMoves(color, index)
				}
				if len(nodeQueue) > 0 {
					index++
					break
				}
			}
		}
		for len(nodeQueue) > 0 {
			searchNode := nodeQueue[0]
			nodeQueue = nodeQueue[1:]
			if !node.inNodeHistory(searchNode) {
				return searchNode
			}
		}
		return nil
	}
}

func (node cNode) String() string {
	sb := strings.Builder{}
	for i := 0; i < 64; i++ {
		if isBit(node.board[pawns][white], i) {
			sb.WriteRune(whitePawn)
		} else if isBit(node.board[kings][white], i) {
			sb.WriteRune(whiteKing)
		} else if isBit(node.board[pawns][black], i) {
			sb.WriteRune(blackPawn)
		} else if isBit(node.board[kings][black], i) {
			sb.WriteRune(blackKing)
		} else {
			// space
			sb.WriteRune('_')
		}
		if (i+1)%8 == 0 {
			// newline after 8 columns
			sb.WriteByte('\n')
		} else {
			sb.WriteByte(' ')
		}
	}
	return sb.String()
}

func cNodeEmpty() cNode {
	return cNode{
		nodeHistory: make(cNodeHistory),
	}
}

func cNodeFullBoard() cNode {
	node := cNodeEmpty()
	// each color has three rows
	// we start with black (since black has positive pawn direction), then white (negative pawn direction)
	for _, row := range []int{0, 1, 2, 5, 6, 7} {
		for i := 0; i < 4; i++ {
			index := row*8 + i*2
			if row%2 != 0 {
				index++
			}
			color := black
			if row > 3 {
				color = white
			}
			node.board[pawns][color] = setBit(node.board[pawns][color], index)
		}
	}
	// add self
	node.addNodeHistory(node)
	return node
}

func (node cNode) cloneNode() cNode {
	// in case of eventually adding more attributes that wont be deep copied
	return node
}

func (node cNode) upgradeToKing(color, index int) cNode {
	if isBit(node.board[pawns][color], index) {
		if (color == black && index >= 56 && index < 64) || (color == white && index >= 0 && index < 8) {
			// upgrade
			clone := node.cloneNode()
			clone.board[pawns][color] = clearBit(clone.board[pawns][color], index)
			clone.board[kings][color] = setBit(clone.board[kings][color], index)
			return clone
		}
	}
	return node
}

func (node cNode) figureMove(figure, color, index, offset int) (bool, cNode) {
	if index < 0 || index > 63 || index+offset < 0 || index+offset > 63 {
		return false, cNode{}
	}
	if !offsetInBoard(index, offset) || node.placeOccupied(index+offset) {
		return false, cNode{}
	}
	clone := node.cloneNode()
	clone.board[figure][color] = clearBit(clone.board[figure][color], index)
	clone.board[figure][color] = setBit(clone.board[figure][color], index+offset)
	return true, clone.upgradeToKing(color, index+offset)
}

func (node cNode) figureJump(figure, color, index, offset int) (bool, cNode) {
	if index < 0 || index > 63 || index+2*offset < 0 || index+2*offset > 63 {
		return false, cNode{}
	}
	if !offsetInBoard(index, offset) || !offsetInBoard(index+offset, offset) {
		return false, cNode{}
	}
	if !node.placeOccupiedColor(enemyColor(color), index+offset) || node.placeOccupied(index+2*offset) {
		return false, cNode{}
	}
	clone := node.cloneNode()
	enemyCol := enemyColor(color)
	clone.board[figure][color] = clearBit(clone.board[figure][color], index)
	clone.board[pawns][enemyCol] = clearBit(clone.board[pawns][enemyCol], index+offset)
	clone.board[kings][enemyCol] = clearBit(clone.board[kings][enemyCol], index+offset)
	clone.board[figure][color] = setBit(clone.board[figure][color], index+2*offset)
	return true, clone.upgradeToKing(color, index+2*offset)
}

// Generate both moves and jumps
func (node cNode) generateFigureMoves(figure, color, index, dir int) []cNode {
	moves := make([]cNode, 0, 2)
	for _, offset := range []int{7, 9} {
		ok, move := node.figureMove(figure, color, index, offset*dir)
		if ok {
			moves = append(moves, move)
		}
		ok, move = node.figureJump(figure, color, index, offset*dir)
		if ok {
			moves = append(moves, move)
		}
	}
	return moves
}

// Generate both moves and jumps
func (node cNode) generatePawnMoves(color, index, dir int) []cNode {
	return node.generateFigureMoves(pawns, color, index, dir)
}

// Generate both moves and jumps
func (node cNode) generateKingMoves(color, index int) []cNode {
	moves := node.generateFigureMoves(kings, color, index, -1)
	return append(moves, node.generateFigureMoves(kings, color, index, 1)...)
}

func (node cNode) placeOccupiedFigureColor(figure, color, index int) bool {
	return isBit(node.board[figure][color], index)
}

func (node cNode) placeOccupiedColor(color, index int) bool {
	return isBit(node.board[pawns][color]|node.board[kings][color], index)
}

func (node cNode) placeOccupied(index int) bool {
	return node.placeOccupiedColor(white, index) || node.placeOccupiedColor(black, index)
}

func (node cNode) inNodeHistory(searchNode cNode) bool {
	nodes, found := node.nodeHistory[searchNode.boardMask()]
	if !found {
		return false
	}
	// multiple boards can have same mask
	for _, n := range nodes {
		if n.board == searchNode.board {
			return true
		}
	}
	return false
}

func (node cNode) addNodeHistory(newNode cNode) {
	mask := newNode.boardMask()
	node.nodeHistory[mask] = append(node.nodeHistory[mask], newNode)
}

func (node cNode) boardMask() uint64 {
	b := &node.board
	return b[pawns][black] | b[kings][black] | b[pawns][white] | b[kings][white]
}

func isBit(num uint64, index int) bool {
	return num&(1<<index) != 0
}

func clearBit(num uint64, index int) uint64 {
	return num &^ (1 << index)
}

func setBit(num uint64, index int) uint64 {
	return clearBit(num, index) | (1 << index)
}

func enemyColor(color int) int {
	if color == white {
		return black
	}
	return white
}

func abs(val int) int {
	if val < 0 {
		return -val
	}
	return val
}

func offsetInBoard(index, offset int) bool {
	return abs(index/8-abs(index+offset)/8)+abs(index%8-abs(index+offset)%8) <= 2
}

func TestCheckersString(t *testing.T) {
	node := cNodeEmpty()
	expected := strings.Repeat("_ _ _ _ _ _ _ _\n", 8)
	if node.String() != expected {
		t.Error("Invalid empty board")
	}
}

func TestCheckersFullBoard(t *testing.T) {
	node := cNodeFullBoard()
	expected := "♙ _ ♙ _ ♙ _ ♙ _\n_ ♙ _ ♙ _ ♙ _ ♙\n♙ _ ♙ _ ♙ _ ♙ _\n_ _ _ _ _ _ _ _" +
		"\n_ _ _ _ _ _ _ _\n_ ♟ _ ♟ _ ♟ _ ♟\n♟ _ ♟ _ ♟ _ ♟ _\n_ ♟ _ ♟ _ ♟ _ ♟\n"
	if node.String() != expected {
		t.Error("Invalid full board")
	}
}

func TestCheckersCustomBoard(t *testing.T) {
	node := cNodeEmpty()
	node.board[pawns][white] = setBit(node.board[pawns][white], 9)
	node.board[kings][white] = setBit(node.board[kings][white], 4)
	node.board[kings][black] = setBit(node.board[kings][black], 33)
	node.board[pawns][black] = setBit(node.board[pawns][black], 35)

	expected := "_ _ _ _ ♛ _ _ _\n_ ♟ _ _ _ _ _ _\n" + strings.Repeat("_ _ _ _ _ _ _ _\n", 2) +
		"_ ♕ _ ♙ _ _ _ _\n" + strings.Repeat("_ _ _ _ _ _ _ _\n", 3)

	if node.String() != expected {
		t.Error("Invalid custom board")
	}
}

func TestCheckersPawnMoves(t *testing.T) {
	{
		// out-of-board top
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 6)
		moves := node.generatePawnMoves(white, 6, -1)
		if len(moves) != 0 {
			t.Error("Cannot go out of board top")
		}
	}
	{
		// out-of-board bottom
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 63)
		moves := node.generatePawnMoves(white, 63, 1)
		if len(moves) != 0 {
			t.Error("Cannot go out of board bottom")
		}
	}
	{
		// out-of-board left
		node := cNodeEmpty()
		node.board[pawns][black] = setBit(0, 8)
		moves := node.generatePawnMoves(black, 8, -1)
		if len(moves) != 1 || !isBit(moves[0].board[pawns][black], 1) {
			t.Error("Must be only one move to top-right")
		}
	}
	{
		// out-of-board right
		node := cNodeEmpty()
		node.board[pawns][black] = setBit(0, 15)
		moves := node.generatePawnMoves(black, 15, -1)
		if len(moves) != 1 || !isBit(moves[0].board[pawns][black], 6) {
			t.Error("Must be only one move to top-left")
		}
	}
	{
		// full move
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 9)
		moves := node.generatePawnMoves(white, 9, -1)
		if len(moves) != 2 {
			t.Error("Must be exactly two moves")
		}
		// pawn has been promoted to king
		board := moves[0].board[kings][white] | moves[1].board[kings][white]
		if !isBit(board, 2) || !isBit(board, 0) {
			t.Error(moves[1])
		}
	}
	{
		node := cNodeEmpty()
		node.board[pawns][black] = setBit(0, 16)
		moves := node.generatePawnMoves(black, 16, 1)
		if len(moves) != 1 {
			t.Error("Must be exactly one move")
		}
		board := moves[0].board[pawns][black]
		if !isBit(board, 25) || isBit(board, 16) {
			t.Error("Invalid moves")
		}
	}
}

func TestCheckersUpgradeToKing(t *testing.T) {
	{
		// white
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 8)
		moves := node.generatePawnMoves(white, 8, -1)
		if len(moves) != 1 {
			t.Error("Must be exactly one move")
		}
		if !isBit(moves[0].board[kings][white], 1) {
			t.Error("Invalid upgrade")
		}
	}
	{
		// black
		node := cNodeEmpty()
		node.board[pawns][black] = setBit(0, 48)
		moves := node.generatePawnMoves(black, 48, 1)
		if len(moves) != 1 {
			t.Error("Must be exactly one move")
		}
		if !isBit(moves[0].board[kings][black], 57) {
			t.Error("Invalid upgrade")
		}
	}
	{
		// jump
		node := cNodeEmpty()
		node.board[pawns][black] = setBit(0, 9)
		node.board[pawns][white] = setBit(0, 16)
		moves := node.generatePawnMoves(white, 16, -1)
		if len(moves) != 1 {
			t.Error("Must be exactly one move")
		}
		if !isBit(moves[0].board[kings][white], 2) {
			t.Error("Invalid jump and upgrade")
		}
	}
}

func TestCheckersPawnJumps(t *testing.T) {
	{
		// out-of-board top
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 10)
		node.board[pawns][black] = setBit(0, 1) | setBit(0, 3)
		moves := node.generatePawnMoves(white, 10, -1)
		if len(moves) != 0 {
			t.Error("Cannot go out of board top")
		}
	}
	{
		// out-of-board bottom
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 54)
		node.board[pawns][black] = setBit(0, 63) | setBit(0, 61)
		moves := node.generatePawnMoves(white, 54, 1)
		if len(moves) != 0 {
			t.Error("Cannot go out of board bottom")
		}
	}
	{
		// out-of-board left
		node := cNodeEmpty()
		node.board[kings][black] = setBit(0, 8) | setBit(0, 10)
		node.board[pawns][white] = setBit(0, 17)
		moves := node.generatePawnMoves(white, 17, -1)
		if len(moves) != 1 {
			t.Error("Must be only one jump to top-right")
		}
		// pawn has been promoted to king
		if !isBit(moves[0].board[kings][black], 8) ||
			isBit(moves[0].board[kings][black], 10) ||
			!isBit(moves[0].board[kings][white], 3) ||
			isBit(moves[0].board[pawns][white], 17) {
			t.Error(moves[0])
		}
	}
	{
		// out-of-board right
		node := cNodeEmpty()
		node.board[kings][black] = setBit(0, 29) | setBit(0, 31)
		node.board[pawns][white] = setBit(0, 22)
		moves := node.generatePawnMoves(white, 22, 1)
		if len(moves) != 1 {
			t.Error("Must be only one jump to top-left")
		}
		if !isBit(moves[0].board[kings][black], 31) ||
			isBit(moves[0].board[kings][black], 29) ||
			!isBit(moves[0].board[pawns][white], 36) ||
			isBit(moves[0].board[pawns][white], 22) {
			t.Error("Invalid jump")
		}
	}
	{
		// casual two jumps both directions
		node := cNodeEmpty()
		node.board[kings][black] = setBit(0, 10) | setBit(0, 12)
		node.board[pawns][white] = setBit(0, 19)
		moves := node.generatePawnMoves(white, 19, -1)
		if len(moves) != 2 {
			t.Error("Expected two jumps")
		}
		// pawn has been promoted to king
		if !isBit(moves[0].board[kings][black], 10) ||
			!isBit(moves[0].board[kings][white], 5) ||
			isBit(moves[0].board[pawns][white], 19) ||
			isBit(moves[0].board[kings][black], 12) {
			t.Error("Invalid move")
		}
		if !isBit(moves[1].board[kings][black], 12) ||
			!isBit(moves[1].board[kings][white], 1) ||
			isBit(moves[1].board[pawns][white], 19) ||
			isBit(moves[1].board[kings][black], 10) {
			t.Error("Invalid move")
		}
	}
	{
		// cannot jump through two figures
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 9) | setBit(0, 18) | setBit(0, 20) | setBit(0, 13)
		node.board[pawns][black] = setBit(0, 27)
		moves := node.generatePawnMoves(black, 27, -1)
		if len(moves) != 0 {
			t.Error("Cannot jump over two figures in a row")
		}
	}
	{
		// cannot jump over own unit
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 17) | setBit(0, 19) | setBit(0, 26)
		moves := node.generatePawnMoves(white, 26, -1)
		if len(moves) != 0 {
			t.Error("Cannot jump over own figure")
		}
	}
}

func TestCheckersKingMovesAndJumps(t *testing.T) {
	{
		node := cNodeEmpty()
		node.board[pawns][white] = setBit(0, 1) | setBit(0, 19)
		node.board[kings][white] = setBit(0, 3) | setBit(0, 17)
		node.board[kings][black] = setBit(0, 10)
		moves := node.generateKingMoves(black, 10)
		if len(moves) != 2 {
			t.Error("Expected two jumps")
		}
		whiteBoard := moves[0].board[pawns][white] | moves[0].board[kings][white]
		if !isBit(whiteBoard, 1) || !isBit(whiteBoard, 3) ||
			isBit(whiteBoard, 17) || !isBit(whiteBoard, 19) ||
			isBit(moves[0].board[kings][black], 10) {
			t.Error("Invalid jump")
		}
		whiteBoard = moves[1].board[pawns][white] | moves[1].board[kings][white]
		if !isBit(whiteBoard, 1) || !isBit(whiteBoard, 3) ||
			!isBit(whiteBoard, 17) || isBit(whiteBoard, 19) ||
			isBit(moves[1].board[kings][black], 10) {
			t.Error("Invalid jump")
		}
	}
	{
		node := cNodeEmpty()
		node.board[kings][white] = setBit(0, 9)
		moves := node.generateKingMoves(white, 9)
		if len(moves) != 4 {
			t.Error("Expected four moves")
		}
		var board uint64
		for i := 0; i < len(moves); i++ {
			board |= moves[i].board[kings][white]
		}
		if isBit(board, 9) || !isBit(board, 0) || !isBit(board, 2) ||
			!isBit(board, 16) || !isBit(board, 18) {
			t.Error("Invalid moves")
		}
	}
}

func TestCheckersNodeHistory(t *testing.T) {
	node := cNodeEmpty()
	if node.inNodeHistory(node) {
		t.Error("Node itself cannot be in history")
	}
	node.addNodeHistory(node)
	if !node.inNodeHistory(node) {
		t.Error("Node itself must be in history")
	}
	node.addNodeHistory(cNodeFullBoard())
	if !node.inNodeHistory(cNodeFullBoard()) {
		t.Error("Expected full board to be in history")
	}
	{
		node1 := cNodeEmpty()
		node1.board[pawns][white] = setBit(0, 33)
		node2 := cNodeEmpty()
		node2.board[kings][black] = setBit(0, 33)
		if node1.boardMask() != node2.boardMask() {
			t.Error("Expected same board mask")
		}
		// same board mask, yet cannot collide in node history
		node.addNodeHistory(node1)
		if !node.inNodeHistory(node1) {
			t.Error("Node expected in hostory")
		}
		if node.inNodeHistory(node2) {
			t.Error("Different node cannot be in history")
		}
	}
}

func TestCheckersCloneNode(t *testing.T) {
	node := cNodeEmpty()
	node.board[kings][white] = setBit(0, 33)
	clone := node.cloneNode()
	if node.boardMask() != clone.boardMask() {
		t.Error("Expected same board mask")
	}
	if node.board != clone.board {
		t.Error("Expected same board")
	}
	if reflect.ValueOf(node.nodeHistory).Pointer() != reflect.ValueOf(clone.nodeHistory).Pointer() {
		t.Error("Nodes should share nodeHistory")
	}
}

func TestCheckersSearchNodeGenerator(t *testing.T) {
	if cNodeEmpty().SearchNodeGenerator()(true) != nil || cNodeEmpty().SearchNodeGenerator()(false) != nil {
		t.Error("Impossible to generate nodes from empty board")
	}
	generator := cNodeFullBoard().SearchNodeGenerator()
	for i := 0; i < 14; i++ {
		if generator(i/7 == 0) == nil {
			t.Error("Should generate more moves")
		}
	}
	if generator(false) != nil || generator(true) != nil {
		t.Error("Impossible to generate more moves")
	}
}

type minimaxFn func(node SearchNode, depth int, maximizing bool) (SearchNode, int)

func TestCheckersMinimaxFullgame(t *testing.T) {
	run := func(maximizing bool, minimax minimaxFn, depth func(bool) int, scoreCheck func(int) bool) {
		sn := cNodeFullBoard()
		// max iterations, should not exceed
		for i := 0; i < 1000 && !sn.IsTerminal(); i++ {
			node, _ := minimax(sn, depth(maximizing), maximizing)
			if node == nil {
				if scoreCheck(sn.Score()) {
					t.Error("bad win")
				}
				return
			}
			sn = node.(cNode)
			sn.addNodeHistory(sn)
			maximizing = !maximizing
		}
		if !sn.IsTerminal() {
			t.Error("Must end in terminal state")
		}
		if scoreCheck(sn.Score()) {
			t.Error("bad win")
		}
	}
	blackScore := func(score int) bool {
		return score <= 0
	}
	whiteScore := func(score int) bool {
		return score >= 0
	}
	blackDepth := func(maximizing bool) int {
		if maximizing {
			// black wins
			return 5
		}
		return 2
	}
	whiteDepth := func(maximizing bool) int {
		return blackDepth(!maximizing)
	}
	run(true, Minimax, blackDepth, blackScore)
	run(false, Minimax, whiteDepth, whiteScore)
	run(true, MinimaxAlphaBetaPrunning, blackDepth, blackScore)
	run(false, MinimaxAlphaBetaPrunning, whiteDepth, whiteScore)

	minimaxConcurrent := func(node SearchNode, depth int, maximizing bool) (SearchNode, int) {
		return MinimaxConcurrent(node, depth, maximizing, 5)
	}
	run(true, minimaxConcurrent, blackDepth, blackScore)
	run(false, minimaxConcurrent, whiteDepth, whiteScore)
}
