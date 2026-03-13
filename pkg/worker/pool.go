package worker

import (
	"context"
	"sync"
)

type Executor func(ctx context.Context, task Task) Result

type Pool struct {
	maxWorkers int
	executor   Executor
}

func NewPool(maxWorkers int, executor Executor) *Pool {
	return &Pool{maxWorkers: maxWorkers, executor: executor}
}

func (p *Pool) Run(ctx context.Context, tasks []Task) []Result {
	if len(tasks) == 0 {
		return nil
	}
	results := make([]Result, len(tasks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.maxWorkers)
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t Task) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = p.executor(ctx, t)
		}(i, task)
	}
	wg.Wait()
	return results
}
