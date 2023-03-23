package workers

import (
	"context"
	"sync"
)

type RemovalTask struct {
	Keys []string
	UUID string
}

type RemovalWorker struct {
	service Remover
	Tasks   <-chan RemovalTask
}

type Remover interface {
	UpdateBatchURL(ctx context.Context, task RemovalTask) error
}

func New(worker Remover, tasks <-chan RemovalTask) *RemovalWorker {
	return &RemovalWorker{
		service: worker,
		Tasks:   tasks,
	}
}

func (w *RemovalWorker) Run(ctx context.Context) error {
	wg := sync.WaitGroup{}

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return ctx.Err()

		case task := <-w.Tasks:
			wg.Add(1)
			go func() {
				defer wg.Done()
				w.Add(ctx, task)
			}()
		}
	}
}

func (w *RemovalWorker) Add(ctx context.Context, task RemovalTask) {
	w.service.UpdateBatchURL(ctx, task)
}
