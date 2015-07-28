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

	outQueue       chan []byte
	requestsWriter io.Writer
}

func NewDiffReporter(config *HTTPOutputConfig) (d *DiffReporter) {
	r := new(DiffReporter)

	r.ignoreErrors = config.diffIgnoreErrors

	if config.diffRequestsFile != "" {
		r.outQueue = make(chan []byte, 100)
		r.requestsWriter = NewFileOutput(config.diffRequestsFile)
		go r.writeDiffs()
	}

	return r
}

func (d *DiffReporter) writeDiffs() {
	for {
		req := <-d.outQueue
		d.requestsWriter.Write(req)
	}
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

	if diffErr != nil {
		if d.ignoreErrors {
			return
		} else {
			log.Println("[DIFF] diffhost error:\n", diffErr)
		}
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
		d.outQueue <- req
	}

}
