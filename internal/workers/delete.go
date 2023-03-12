package workers

import (
	"context"
	"sync"
)

type DeleteTask struct {
	Keys []string
	UUID string
}

type DeleteWorker struct {
	service Deleter
	Tasks   <-chan DeleteTask
}

type Deleter interface {
	UpdateBatchURL(ctx context.Context, task DeleteTask) error
}

func NewWorker(worker Deleter, tasks <-chan DeleteTask) *DeleteWorker {
	return &DeleteWorker{
		service: worker,
		Tasks:   tasks,
	}
}

func (w *DeleteWorker) RunWorker(ctx context.Context) {
	wg := sync.WaitGroup{}

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			return

		case task := <-w.Tasks:
			wg.Add(1)
			go func() {
				defer wg.Done()
				w.AddWorker(ctx, task)
			}()
		}
	}
}

func (w *DeleteWorker) AddWorker(ctx context.Context, task DeleteTask) {
	w.service.UpdateBatchURL(ctx, task)
}
