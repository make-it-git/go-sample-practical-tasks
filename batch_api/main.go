package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"example/internal/http/models"
	"example/internal/worker"
)

func main() {
	const (
		numWorkers = 3
		batchSize  = 10
	)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	ctx, cancel := context.WithCancel(context.Background())
	srv := http.Server{
		Addr: "127.0.0.1:8000",
	}

	go func() {
		defer func() {
			if err := srv.Shutdown(ctx); err != nil {
				logger.Warn("HTTP server Shutdown", "error", err)
			}
			logger.Debug("HTTP server stopped")
		}()
		defer cancel()

		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

		select {
		case <-signalCh:
			logger.Warn("Got signal, stoppping")
		case <-ctx.Done():
			logger.Warn("Context cancelled, stopping")
		}
	}()

	var wg sync.WaitGroup
	wg.Add(numWorkers)
	batchWorker := worker.NewBatchWorker(ctx, logger, &wg, numWorkers, batchSize)

	http.HandleFunc("/process", process(batchWorker))

	logger.Info(fmt.Sprintf("Start listening at %s", srv.Addr))
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		logger.Error("HTTP server Shutdown", "error", err)
		cancel()
	}

	wg.Wait()
}

func process(batchWorker *worker.BatchWorker) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		var task models.Request
		err := json.NewDecoder(req.Body).Decode(&task)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		ch, err := batchWorker.Enqueue(&worker.Task{ID: task.Id})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ProcessResponseError{Error: err.Error()})
			return
		}

		result := <-ch

		if result.Error != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(models.ProcessResponseError{Error: result.Error.Error()})
			return
		}

		response := fmt.Sprintf("Request with id %d processed with result %s", task.Id, result.Response)
		json.NewEncoder(w).Encode(models.ProcessResponse{Response: response})
	}
}
