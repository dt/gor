package main

import (
	"bytes"
	"io"
	"log"
	"sync/atomic"
	"time"

	"github.com/buger/gor/proto"
)

type DiffReporter struct {
	totalDiffs int64

	ignoreErrors bool

	requestsWriter io.Writer
}

func NewDiffReporter(config *HTTPOutputConfig) (d *DiffReporter) {
	r := new(DiffReporter)

	r.ignoreErrors = config.diffIgnoreErrors

	if config.diffRequestsFile != "" {
		r.requestsWriter = NewFileOutput(config.diffRequestsFile)
	}

	return r
}

func (d *DiffReporter) ResponseAnalyze(client *HTTPClient, req, resp []byte, rtt int64, err error) {
	// Bail early if ignoring errors and original response is an error.
	if d.ignoreErrors && err != nil {
		return
	}

	start := time.Now()
	diffResp, diffErr := client.Send(req)
	stop := time.Now()

	if err != nil && diffErr != nil {
		return
	}

	if bytes.Equal(resp, diffResp) {
		return
	}

	if d.ignoreErrors && diffErr != nil {
		return
	}

	if !d.ignoreErrors {
		log.Println("[DIFF] diffhost error:\n", diffErr)
	}

	diffRtt := RttDurationToMs(stop.Sub(start))

	atomic.AddInt64(&d.totalDiffs, 1)

	diffNum := atomic.LoadInt64(&d.totalDiffs)

	// TODO(davidt): Log to file, track p99, etc
	respSize := len(resp)
	diffSize := len(diffResp)

	log.Printf("[DIFF %d] %s %s status: %s v %s size: %d v %d (%d) time: %dms vs %dms (%d)",
		diffNum,
		proto.Method(req),
		proto.Path(req),
		proto.Status(resp),
		proto.Status(diffResp),
		respSize,
		diffSize,
		respSize-diffSize,
		rtt,
		diffRtt,
		rtt-diffRtt,
	)

	if d.requestsWriter != nil {
		d.requestsWriter.Write(req)
	}

}
