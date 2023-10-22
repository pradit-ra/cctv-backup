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

func (c *cctvBackup) Backup(items []SearchMatchItem) error {
	g := &errgroup.Group{}
	for _, it := range items {
		// generate unique file name (key)
		key := buildKey(it.PlaybackURI)
		reqPayload := bytes.NewReader([]byte(getDownloadData(it.PlaybackURI)))
		// goroutine to copy multiple video files into a storage simultaneously
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
			logger.Info("start downloading video", "key", key)
			req := newCCTVRequest(downloadEndpoint(c.info.HostURL), reqPayload)

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

func getDownloadData(RTSPEndpoint string) string {
	tmpl := func(name, t string) *template.Template {
		return template.Must(template.New(name).Parse(t))
	}
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
		RTSPEndpoint: RTSPEndpoint,
	})
	return compiledTmpl.String()
}

func buildKey(RTSPEndpoint string) string {
	u, _ := url.Parse(RTSPEndpoint)
	name := u.Query().Get("name")
	return fmt.Sprintf("%s.h264", name)
}
