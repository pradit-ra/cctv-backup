package main

import (
	"cctv-backup/v1/cctv"
	"cctv-backup/v1/worker"
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// package scoped variable
var (
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	//Channel to listen for signals.
	signalChan chan (os.Signal)               = make(chan os.Signal, 1)
	metricChan chan (*worker.ExecutionMetric) = make(chan *worker.ExecutionMetric)

	// cli options
	numOfWorker        = 5
	taskBuffer         = 10 //Goroutine (user thread) should be configurable
	bucket      string = ""
	prefix      string = "/Users/pradit.ra/TDProjects/cctv-backup/temp/downloads"
	datafile    string = "backup.yaml"
)

// Yaml Unmarshal
type BackupData struct {
	ChannelID  string `yaml:"channelID"`
	HostURL    string `yaml:"hostURL"`
	Credential struct {
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	} `yaml:"credential"`
	Backup struct {
		From string `yaml:"from"`
		To   string `yaml:"to"`
	} `yaml:"backup"`
}

var errorHandler worker.FailedTaskHandlerFunc = func(err error, info *cctv.CCTVInfo) {
	logger.Warn("Error to backup Video CCTV", "channelID", info.ChannelID, "url", info.HostURL.Host, "err", err)
}

func loadTasks() ([]worker.Task, error) {
	var backupdata []BackupData
	var tasks []worker.Task

	f, err := os.ReadFile(datafile)
	if err != nil {
		return nil, fmt.Errorf("error to read data file %w", err)
	}
	err = yaml.Unmarshal(f, &backupdata)
	if err != nil {
		return nil, fmt.Errorf("error to unmarshal data %w", err)
	}
	logger.Debug(fmt.Sprintf("load data file contains %d records", len(backupdata)))
	for _, d := range backupdata {
		h, err := url.Parse(d.HostURL)
		if err != nil {
			return nil, fmt.Errorf("error to parse host url %w", err)
		}
		// storage, err := cctv.NewGCSBackupStorage(bucket, prefix)
		storage := cctv.NewFileBackupStorage(prefix)
		if err != nil {
			return nil, fmt.Errorf("error to create storage %w", err)
		}
		cc := cctv.NewCCTVBackup(d.ChannelID, h, &cctv.Credential{
			User:     d.Credential.User,
			Password: d.Credential.Password,
		}, storage)
		t := worker.NewCCTVBackupTask(cc, worker.BackupPayload{
			SearchFrom: d.Backup.From,
			SearchTo:   d.Backup.To,
		}, errorHandler)
		tasks = append(tasks, t)
	}
	return tasks, nil
}

func main() {
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	// start profiling goroutine
	// go logGoroutine()

	logger.Info("Run CCTV video backup job")
	ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	wp := worker.NewWorkerPool(numOfWorker, taskBuffer, metricChan)

	tasks, err := loadTasks()
	if err != nil {
		logger.Error("error to load data file", "err", err)
		os.Exit(1)
	}
	// Start and Wait
	wp.StartWorker(ctx)

	// Receive output from signalChan or pool execution metric channel
	go func() {
		for {
			select {
			case sig := <-signalChan:
				logger.Warn(fmt.Sprintf("%s signal caught", sig))
				// send signal to context for stop worker gracefully
				cancel()
			case metric, ok := <-metricChan:
				if ok {
					logger.Debug("receive job execution metric",
						"workerID", metric.WorkerID,
						"start", metric.Start.Format(time.RFC3339),
						"end", metric.End.Format(time.RFC3339),
						"elapsed", metric.Elapsed.String(),
					)
				}
			}
		}
	}()
	// run and wait
	wp.RunTasks(tasks...)
}

func logGoroutine() {
	for {
		logger.Debug(fmt.Sprintf("[Goroutine] Total current goroutine: %d", runtime.NumGoroutine()))
		time.Sleep(5 * time.Second)
	}
}
