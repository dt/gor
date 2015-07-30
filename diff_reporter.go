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

func isErr(err error, resp []byte) bool {
	return err != nil || proto.Status(resp)[0] == '5'
}

func (d *DiffReporter) ResponseAnalyze(client *HTTPClient, req, respA []byte, rttA time.Duration, rawErrA error) {

	GorMetrics.Inc("diffing.total")

	start := time.Now()
	respB, rawErrB := client.Send(req)
	stop := time.Now()
	rttB := stop.Sub(start)

	errA := isErr(rawErrA, respA)
	errB := isErr(rawErrB, respB)

	if errA {
		GorMetrics.Inc("diffing.err.a")
	} else {
		GorMetrics.Timing("diffing.rtt.a", rttA)
	}

	if errB {
		GorMetrics.Inc("diffing.err.b")
	} else {
		GorMetrics.Timing("diffing.rtt.b", rttB)
	}

	if (errA && errB) || (d.ignoreErrors && (errA || errB)) {
		return
	}

	if bytes.Equal(respA[proto.MIMEHeadersEndPos(respA):], respB[proto.MIMEHeadersEndPos(respB):]) {
		GorMetrics.Inc("diffing.match")
		return
	}

	GorMetrics.Inc("diffing.diff")

	atomic.AddInt64(&d.totalDiffs, 1)

	diffNum := atomic.LoadInt64(&d.totalDiffs)

	sizeA := len(respA)
	sizeB := len(respB)

	log.Printf("[DIFF %d] %s %s status: %s v %s size: %d v %d (%d) time: %dms vs %dms (%d)",
		diffNum,
		proto.Method(req),
		proto.Path(req),
		proto.Status(respA),
		proto.Status(respB),
		sizeA,
		sizeB,
		sizeA-sizeB,
		DurationToMs(rttA),
		DurationToMs(rttB),
		DurationToMs(rttA-rttB),
	)

	if d.requestsWriter != nil {
		d.outQueue <- req
	}

}
