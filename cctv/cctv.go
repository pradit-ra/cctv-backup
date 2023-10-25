package cctv

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/icholy/digest"
)

var logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

const XMLContentType = "application/xml"

func newHttpClient(user, password string) *http.Client {
	return &http.Client{
		// donwload timeout
		// TODO: configurable
		Timeout: 5 * time.Minute,
		Transport: &digest.Transport{
			Username: user,
			Password: password,
		},
	}
}

func newCCTVRequest(urlEndpoint *url.URL, body io.Reader) *http.Request {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, urlEndpoint.String(), body)
	if err != nil {
		logger.Error("Create CCTV POST XML request", "err", err.Error())
	}
	req.Header.Set("Content-Type", XMLContentType)
	return req
}

type Credential struct {
	User, Password string
}

type CCTVInfo struct {
	TrackID  string
	HostAddr string
}

type TimeSegment struct {
	Start time.Time
	End   time.Time
}

type CCTVBackup interface {
	Backup(segments []TimeSegment) error
	GetInfo() *CCTVInfo
}

// implementation of CCTVBackup interface
type cctvBackup struct {
	info    *CCTVInfo
	httpC   *http.Client
	storage BackupStorage
}

func (c *cctvBackup) GetInfo() *CCTVInfo {
	return c.info
}

func NewCCTVBackup(trackID string, hostAddr string, cred *Credential, storage BackupStorage) CCTVBackup {
	return &cctvBackup{
		info: &CCTVInfo{
			TrackID:  trackID,
			HostAddr: hostAddr,
		},
		httpC:   newHttpClient(cred.User, cred.Password),
		storage: storage,
	}
}
