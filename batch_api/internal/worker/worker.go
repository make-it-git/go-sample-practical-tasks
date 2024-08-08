package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type worker struct {
	logger *slog.Logger
	ID     int
	queue  chan *queueItem
	mu     *sync.Mutex
}

func newWorker(logger *slog.Logger, id int, batchSize int64) *worker {
	queue := make(chan *queueItem, batchSize)
	return &worker{
		logger: logger,
		ID:     id,
		queue:  queue,
		mu:     &sync.Mutex{},
	}
}

func (w *worker) start(ctx context.Context, wg *sync.WaitGroup, batchSize int) {
	defer wg.Done()

	var queueItems []*queueItem
	var queueItem *queueItem
	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			w.withLock(func() {
				w.processBatch(queueItems)
				queueItems = nil
			})
			return
		case queueItem = <-w.queue:
			w.logger.Debug("worker got task", slog.Int("workerID", w.ID), slog.Int("taskID", queueItem.task.ID))
			w.withLock(func() {
				queueItems = append(queueItems, queueItem)
				if len(queueItems) >= batchSize {
					w.processBatch(queueItems)
					queueItems = nil
				}
			})
		case <-timer.C:
			w.withLock(func() {
				w.processBatch(queueItems)
				queueItems = nil
			})
		}
	}
}

func (w *worker) withLock(f func()) {
	w.mu.Lock()
	defer w.mu.Unlock()
	f()
}

func (w *worker) TrySend(task *Task) (<-chan Result, bool) {
	if !w.mu.TryLock() {
		return nil, false
	}
	defer w.mu.Unlock()

	ch := make(chan Result)
	w.queue <- &queueItem{task: *task, ch: ch}

	return ch, true
}

func (w *worker) processBatch(queueItems []*queueItem) {
	if len(queueItems) == 0 {
		return
	}

	w.logger.Info("worker processing batch", slog.Int("workerID", w.ID), slog.Int("batchSize", len(queueItems)))
	time.Sleep(time.Second * 5)
	w.logger.Info("worker processed batch", slog.Int("workerID", w.ID), slog.Int("batchSize", len(queueItems)))
	// do actual http request
	for _, item := range queueItems {
		item.ch <- Result{
			Response: fmt.Sprintf("Result of task %d", item.task.ID),
			Error:    nil,
		}
		close(item.ch)
	}
}
