package cctv

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"text/template"

	"golang.org/x/sync/errgroup"
)

const dtfmt = "20060102T150405"

func (c *cctvBackup) Backup(segments []TimeSegment) error {
	g := &errgroup.Group{}
	for _, s := range segments {
		key := buildKey(s)
		reqPayload := bytes.NewReader([]byte(buildData(c.info.HostAddr, c.info.TrackID, s)))
		// goroutine to copy multiple video segments into a storage simultaneously
		g.Go(func() error {
			var reader io.ReadCloser
			// handle stream close gracefully after function is called
			defer func() {
				if reader != nil {
					if err := reader.Close(); err != nil {
						logger.Error("Close Http Response body reader", "err", err.Error())
					}
				}
			}()
			downloadURL := &url.URL{Scheme: "http", Host: c.info.HostAddr, Path: "/ISAPI/ContentMgmt/download"}
			logger.Info("start downloading video", "key", key, "downloadURL", downloadURL.String())
			req := newCCTVRequest(downloadURL, reqPayload)

			resp, err := c.httpC.Do(req)
			if err != nil {
				return fmt.Errorf("send http POST request, %w", err)
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("HTTP request returns not 200 ok, key: %s, return code: %d", key, resp.StatusCode)
			}
			reader = resp.Body

			return c.storage.Write(key, reader)
		})
	}
	// Wait for captured videos are uploaded into GCS
	return g.Wait()
}

func buildData(hostAddr string, trackID string, s TimeSegment) string {
	tmpl := func(name, t string) *template.Template {
		return template.Must(template.New(name).Parse(t))
	}
	rtspUrlData := fmt.Sprintf("rtsp://%s/Streaming/tracks/%s?starttime=%s&endtime=%s", hostAddr, trackID, s.Start, s.End)
	xml := tmpl("xml", `
	<?xml version="1.0" encoding="UTF-8" ?>
	<downloadRequest version="1.0" xmlns="http://www.isapi.org/ver20/XMLSchema">
	  <playbackURI>"{{ .RTSPEndpoint }}"</playbackURI>
	</downloadRequest>
	`)
	var compiledTmpl bytes.Buffer
	xml.Execute(&compiledTmpl, struct {
		RTSPEndpoint string
	}{
		RTSPEndpoint: rtspUrlData,
	})
	return compiledTmpl.String()
}

func buildKey(seg TimeSegment) string {
	return fmt.Sprintf("%v-%v.h264", seg.Start.Format(dtfmt), seg.End.Format(dtfmt))
}
