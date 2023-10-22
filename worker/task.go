package worker

import (
	"cctv-backup/v1/cctv"
	"fmt"

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

type BackupPayload struct {
	SearchFrom string
	SearchTo   string
}

func NewCCTVBackupTask(cctvBk cctv.CCTVBackup, payload BackupPayload, handler FailedTaskHandlerFunc) Task {
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
	payload           BackupPayload
	failedTaskHandler FailedTaskHandlerFunc
}

func (c *cctvBackupTask) Exec() error {
	sr, err := c.cctvBk.SearchVideo(c.payload.SearchFrom, c.payload.SearchTo)
	if err != nil {
		return err
	}
	logger.Info("Done search video in range", "from", c.payload.SearchFrom, "to", c.payload.SearchTo, "found", len(sr.SearchMatchItems))
	if err := c.cctvBk.Backup(sr.SearchMatchItems); err != nil {
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
