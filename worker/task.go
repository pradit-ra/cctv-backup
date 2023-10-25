package worker

import (
	"cctv-backup/v1/cctv"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type FailedTaskHandlerFunc func(error, *cctv.CCTVInfo)

type Task interface {
	GetID() string
	// Exec perform task execution
	Exec() error
	// onFailure to handle any error returned from Exec
	onFailure(error)
}

type TaskPayload struct {
	Segments []struct {
		From time.Time
		To   time.Time
	}
}

func NewCCTVBackupTask(cctvBk cctv.CCTVBackup, payload TaskPayload, handler FailedTaskHandlerFunc) Task {
	return &cctvBackupTask{
		id:                uuid.NewString(),
		cctvBk:            cctvBk,
		payload:           payload,
		failedTaskHandler: handler,
	}
}

// Implementation of Task
type cctvBackupTask struct {
	id                string
	cctvBk            cctv.CCTVBackup
	payload           TaskPayload
	failedTaskHandler FailedTaskHandlerFunc
}

func (c *cctvBackupTask) Exec() error {
	var segs []cctv.TimeSegment
	for _, p := range c.payload.Segments {
		segs = append(segs, cctv.TimeSegment{
			Start: p.From,
			End:   p.To,
		})
	}
	if err := c.cctvBk.Backup(segs); err != nil {
		return fmt.Errorf("Task exec error %w", err)
	}
	return nil
}

func (c *cctvBackupTask) onFailure(err error) {
	c.failedTaskHandler(err, c.cctvBk.GetInfo())
}

func (c *cctvBackupTask) SetID(id string) {
	c.id = id
}

func (c *cctvBackupTask) GetID() string {
	return c.id
}
