package queue

import (
	"context"
	"sync"
)

type Job struct {
	Index int
	Data  interface{}
}

type Worker func(Job) (interface{}, error)

type Result struct {
	Index int
	Value interface{}
	Err   error
}

type Queue struct {
	jobs    chan Job
	results chan Result
	workers int
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

func New(workers int, fn Worker) *Queue {
	ctx, cancel := context.WithCancel(context.Background())
	q := &Queue{
		jobs:    make(chan Job, 128),
		results: make(chan Result, 128),
		workers: workers,
		cancel:  cancel,
	}

	for range workers {
		q.wg.Add(1)
		go func() {
			defer q.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-q.jobs:
					if !ok {
						return
					}
					val, err := fn(job)
					q.results <- Result{Index: job.Index, Value: val, Err: err}
					if err != nil {
						cancel()
						return
					}
				}
			}
		}()
	}

	return q
}

func (q *Queue) Add(job Job) {
	q.jobs <- job
}

func (q *Queue) Wait() []Result {
	close(q.jobs)
	q.wg.Wait()
	close(q.results)

	var results []Result
	for r := range q.results {
		results = append(results, r)
	}
	return results
}
