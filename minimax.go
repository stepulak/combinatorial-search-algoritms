package csa

import (
	"math"
)

type SearchNodeGenerator func(maximizing bool) SearchNode

type SearchNode interface {
	Score() int
	IsTerminal() bool
	SearchNodeGenerator() SearchNodeGenerator
}

func Minimax(node SearchNode, depth int, maximizing bool) (SearchNode, int) {
	if depth == 0 || node.IsTerminal() {
		return node, node.Score()
	}
	// default minimizing player
	var bestNode SearchNode
	bestScore := MinimaxInitScore(maximizing)
	for generator := node.SearchNodeGenerator(); ; {
		childNode := generator(maximizing)
		if childNode == nil {
			break
		}
		_, newScore := Minimax(childNode, depth-1, !maximizing)
		if (maximizing && newScore >= bestScore) || (!maximizing && newScore <= bestScore) {
			bestScore = newScore
			bestNode = childNode
		}
	}
	return bestNode, bestScore
}

func MinimaxAlphaBetaPrunning(node SearchNode, depth int, maximizing bool) (SearchNode, int) {
	var alpha, beta int
	alpha, beta = math.MinInt, math.MaxInt
	return minimaxAlphaBetaPrunningImpl(node, depth, alpha, beta, maximizing)
}

func minimaxAlphaBetaPrunningImpl(node SearchNode, depth, alpha, beta int, maximizing bool) (SearchNode, int) {
	if depth <= 0 || node.IsTerminal() {
		return node, node.Score()
	}
	// default minimizing player
	var bestNode SearchNode
	bestScore := MinimaxInitScore(maximizing)
	for generator := node.SearchNodeGenerator(); ; {
		childNode := generator(maximizing)
		if childNode == nil {
			break
		}
		_, newScore := minimaxAlphaBetaPrunningImpl(childNode, depth-1, alpha, beta, !maximizing)
		if maximizing {
			if newScore > alpha {
				alpha = newScore
				bestNode = childNode
				bestScore = newScore
			}
		} else {
			if newScore < beta {
				beta = newScore
				bestNode = childNode
				bestScore = newScore
			}
		}
		if alpha >= beta {
			break
		}
	}
	return bestNode, bestScore
}

func MinimaxInitScore(maximizing bool) int {
	if maximizing {
		return math.MinInt
	}
	return math.MaxInt
}
