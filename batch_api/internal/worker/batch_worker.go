package worker

import (
	"context"
	"log/slog"
	"sync"
)

type BatchWorker struct {
	workers []worker
	logger  *slog.Logger
}

func NewBatchWorker(ctx context.Context, logger *slog.Logger, wg *sync.WaitGroup, numWorkers int, batchSize int) *BatchWorker {
	var workers []worker
	for i := 1; i <= numWorkers; i++ {
		worker := newWorker(logger, i, int64(batchSize))
		workers = append(workers, *worker)
		go worker.start(ctx, wg, batchSize)
	}
	return &BatchWorker{
		workers: workers,
		logger: logger,
	}
}

func (b *BatchWorker) Enqueue(task *Task) (<-chan Result, error) {
	for _, w := range b.workers {
		if ch, ok := w.TrySend(task); ok {
			b.logger.Info("sent task", slog.Int("taskID", task.ID), slog.Int("workerID", w.ID))
			return ch, nil
		}

		b.logger.Warn("skip busy worker", slog.Int("workerID", w.ID))
	}

	return nil, ErrNotQueued
}
