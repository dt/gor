package main

import (
	"github.com/rcrowley/go-metrics"
	"log"
	"strconv"
	"time"
)

type MetricsConfig struct {
}

var GorMetrics GorMetricsTracker

type GorMetricsTracker struct {
	registry metrics.Registry
}

func (t *GorMetricsTracker) Inc(stat string) {
	metrics.GetOrRegisterCounter(stat, t.registry).Inc(1)
}

func (t *GorMetricsTracker) Timing(stat string, d time.Duration) {
	metrics.GetOrRegisterTimer(stat, t.registry).Update(d)
}

func InitMetricsManager(c *MetricsConfig) {
	GorMetrics.registry = metrics.NewRegistry()
}

const (
	rate = 5
)

type GorStat struct {
	statName string
	latest   int
	mean     int
	max      int
	count    int
}

func NewGorStat(statName string) (s *GorStat) {
	s = new(GorStat)
	s.statName = statName
	s.latest = 0
	s.mean = 0
	s.max = 0
	s.count = 0

	if Settings.stats {
		log.Println(s.statName + ":latest,mean,max,count,count/second")
		go s.reportStats()
	}
	return
}

func (s *GorStat) Write(latest int) {
	if Settings.stats {
		if latest > s.max {
			s.max = latest
		}
		if latest != 0 {
			s.mean = (s.mean + latest) / 2
		}
		s.latest = latest
		s.count = s.count + 1
	}
}

func (s *GorStat) Reset() {
	s.latest = 0
	s.max = 0
	s.mean = 0
	s.count = 0
}

func (s *GorStat) String() string {
	return s.statName + ":" + strconv.Itoa(s.latest) + "," + strconv.Itoa(s.mean) + "," + strconv.Itoa(s.max) + "," + strconv.Itoa(s.count) + "," + strconv.Itoa(s.count/rate)
}

func (s *GorStat) reportStats() {
	for {
		log.Println(s)
		s.Reset()
		time.Sleep(rate * time.Second)
	}
}
