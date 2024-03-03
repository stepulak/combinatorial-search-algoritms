package csa

func MinimaxConcurrent(node SearchNode, depth int, maximizing bool, workers int) (SearchNode, int) {
	if depth == 0 || node.IsTerminal() {
		return node, node.Score()
	}
	// setup workers
	jobs := make(chan workerJob, workers*5)
	results := make(chan workerResult, workers*5)
	for i := 0; i < workers; i++ {
		go minimaxConcurrentWorker(jobs, results)
	}
	// feed workers
	totalJobs := make(chan int)
	go minimaxConcurrentFeeder(node, depth, maximizing, jobs, totalJobs)
	// consume results
	return minimaxConcurrentConsumer(maximizing, jobs, results, totalJobs)
}

type workerJob struct {
	id         int
	node       SearchNode
	depth      int
	maximizing bool
}

type workerResult struct {
	jobId int
	node  SearchNode
	score int
}

func minimaxConcurrentWorker(jobs <-chan workerJob, results chan<- workerResult) {
	for job := range jobs {
		_, score := MinimaxAlphaBetaPrunning(job.node, job.depth, job.maximizing)
		results <- workerResult{job.id, job.node, score}
	}
}

func minimaxConcurrentFeeder(node SearchNode, depth int, maximizing bool, jobs chan<- workerJob, totalJobs chan<- int) {
	counter := 0
	for generator := node.SearchNodeGenerator(); ; {
		childNode := generator(maximizing)
		if childNode == nil {
			totalJobs <- counter
			return
		}
		jobs <- workerJob{counter, childNode, depth - 1, !maximizing}
		counter++
	}
}

func minimaxConcurrentConsumer(maximizing bool, jobs chan workerJob, results chan workerResult, totalJobs chan int) (SearchNode, int) {
	best := workerResult{-1, nil, MinimaxInitScore(maximizing)}
	numResults, numJobs := 0, -1
	for {
		select {
		case result := <-results:
			if (maximizing && result.score > best.score) || (!maximizing && result.score < best.score) {
				best = result
			}
			if result.score == best.score && result.jobId < best.jobId {
				// if not ordered by jobId, we could get non-deterministic results
				best = result
			}
			numResults++
		case numJobs = <-totalJobs:
			// all jobs have been assigned
		}
		if numJobs >= 0 && numResults == numJobs {
			// all jobs have been assigned and also finished successfully
			close(jobs)
			close(results)
			close(totalJobs)
			return best.node, best.score
		}
	}
}
