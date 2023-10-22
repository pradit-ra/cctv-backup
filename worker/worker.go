package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// type WorkerDoneFunc = func(jobNo int) bool

var (
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
)

type WorkerPool interface {
	// Start runner all pools to serve incoming job
	StartWorker(ctx context.Context)

	// AddTask to add new task into worker pool queue
	RunTasks(...Task)
}

type workerPool struct {
	// number of worker will be sprawn1
	numOfWorker int

	// task channel
	taskChan chan Task

	// enaure Start() is called only once
	onceStart sync.Once

	onceClose sync.Once
	// channel to sending job exeuction metric
	metricChan chan *ExecutionMetric

	wg *sync.WaitGroup
}

type ExecutionMetric struct {
	WorkerID int
	Start    time.Time
	End      time.Time
	Elapsed  time.Duration
}

func NewWorkerPool(n int, taskBuffer int, metricChan chan *ExecutionMetric) WorkerPool {
	return &workerPool{
		numOfWorker: n,
		metricChan:  metricChan,
		taskChan:    make(chan Task, taskBuffer),
		onceStart:   sync.Once{},
		onceClose:   sync.Once{},
		wg:          &sync.WaitGroup{},
	}
}

func (w *workerPool) StartWorker(ctx context.Context) {
	w.onceStart.Do(func() {
		logger.Info("Starting worker pool", "poolsize", w.numOfWorker)
		for i := 1; i <= w.numOfWorker; i++ {
			go w.work(ctx, i)
		}
	})
}

// Add task into task queue. The task will be sent to worker pool for execution
// This will wait until all task done
func (w *workerPool) RunTasks(tasks ...Task) {
	w.wg.Add(len(tasks))
	// non-blocking add task
	go func() {
		for _, t := range tasks {
			w.taskChan <- t
		}
	}()
	w.wg.Wait()
}

func (w *workerPool) work(ctx context.Context, ID int) {
	logger.Info(fmt.Sprintf("woker %d/%d is running", ID, w.numOfWorker))

	for {
		select {
		case <-ctx.Done():
			logger.Warn("worker was interrupted by context cancel", "worker", ID)
			if ctx.Err() != nil {
				logger.Error("Cancel with error", "err", ctx.Err())
			}
			w.closeTaskQueue()
		case t, ok := <-w.taskChan:
			if !ok {
				logger.Warn("stopping worker with closed task queue channel", "worker", ID)
				w.wg.Done()
				return
			}
			logger.Debug("Task run profile", "jobID", t.GetID(), "wokerID", ID)
			var m ExecutionMetric
			m.WorkerID = ID
			m.Start = time.Now()

			if err := t.Exec(); err != nil {
				t.onFailure(err)
			}

			m.End = time.Now()
			m.Elapsed = time.Since(m.Start)
			if w.metricChan != nil {
				w.metricChan <- &m
			}
			w.wg.Done()
		}
	}
}

func (w *workerPool) closeTaskQueue() {
	w.onceClose.Do(func() {
		close(w.taskChan)
	})
}
