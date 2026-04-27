package worker

import (
	"context"
	"sync"
)

type IdempotencyChecker interface {
	// CheckAndMark atomically checks whether key was processed and marks it as
	// processed in a single operation. Returns true if the key is new (not yet
	// processed), false if it was already present. Use this instead of separate
	// IsProcessed + MarkProcessed calls to avoid TOCTOU races.
	CheckAndMark(ctx context.Context, key string) (bool, error)
	IsProcessed(ctx context.Context, key string) (bool, error)
	MarkProcessed(ctx context.Context, key string) error
}

type Job func(ctx context.Context) error

type Pool struct {
	jobs    chan Job
	wg      sync.WaitGroup
	workers int
}

func NewPool(workers int, queueSize int) *Pool {
	return &Pool{
		jobs:    make(chan Job, queueSize),
		workers: workers,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := range p.workers {
		_ = i
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case job, ok := <-p.jobs:
					if !ok {
						return
					}
					_ = job(ctx)
				}
			}
		}()
	}
}

func (p *Pool) Submit(ctx context.Context, job Job) error {
	select {
	case p.jobs <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) Stop() {
	close(p.jobs)
	p.wg.Wait()
}
