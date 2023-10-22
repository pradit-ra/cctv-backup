package cctv

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"text/template"

	"github.com/google/uuid"
)

func (c *cctvBackup) SearchVideo(start string, end string) (SearchResult, error) {
	var sr SearchResult
	reqPayload := bytes.NewReader([]byte(getSearchData(c.info.ChannelID, start, end)))
	req := newCCTVRequest(searchEndpoint(c.info.HostURL), reqPayload)
	resp, err := c.httpC.Do(req)
	if err != nil {
		return sr, fmt.Errorf("send http POST request, %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return sr, fmt.Errorf("HTTP request returns not 200 ok, return code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return sr, fmt.Errorf("read response body, %w", err)
	}

	if err := xml.Unmarshal(body, &sr); err != nil {
		return sr, fmt.Errorf("unmarshal Search result, %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.Error("Close Http Response body reader", "err", err.Error())
		}
	}()
	return sr, nil
}

func getSearchData(channelID string, start string, end string) string {
	searchID := uuid.NewString()
	tmpl := func(name, t string) *template.Template {
		return template.Must(template.New(name).Parse(t))
	}
	xml := tmpl("xml", `
		<?xml version="1.0" encoding="UTF-8" ?>
		<CMSearchDescription>
			<searchID>{{ .SearchID }}</searchID>
		<trackList>
			<trackID>{{ .ChannelID }}</trackID>
		</trackList>
		<timeSpanList>
			<timeSpan>
				<startTime>{{ .Start }}</startTime>
				<endTime>{{ .End }}</endTime>
			</timeSpan>
		</timeSpanList>
		<contentTypeList>
			<contentType>video</contentType>
		</contentTypeList>
		<maxResults>100</maxResults>
		<searchResultPostion>0</searchResultPostion>
		<metadataList>
			<metadataDescriptor>recordType.meta.std-cgi.com</metadataDescriptor>
		</metadataList>
		</CMSearchDescription>
	`)
	var compiledTmpl bytes.Buffer
	xml.Execute(&compiledTmpl, struct {
		SearchID  string
		ChannelID string
		Start     string
		End       string
	}{
		SearchID:  searchID,
		ChannelID: channelID,
		Start:     start,
		End:       end,
	})
	return compiledTmpl.String()
}

type SearchResult struct {
	XMLName          xml.Name          `xml:"CMSearchResult"`
	SearchID         string            `xml:"searchID"`
	SearchMatchItems []SearchMatchItem `xml:"matchList>searchMatchItem"`
}

type SearchMatchItem struct {
	TrackID     int    `xml:"trackID"`
	ContentType string `xml:"mediaSegmentDescriptor>contentType"`
	CodecType   string `xml:"mediaSegmentDescriptor>codecType"`
	PlaybackURI string `xml:"mediaSegmentDescriptor>playbackURI"`
}
