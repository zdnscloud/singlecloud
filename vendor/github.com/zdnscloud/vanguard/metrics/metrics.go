package metrics

import (
	"net/http"
	"time"

	"github.com/zdnscloud/g53"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zdnscloud/vanguard/core"
)

var gMetrics = newMetrics()

const TotalView = "any"

type Metrics struct {
	reg      *prometheus.Registry
	viewQps  map[string]*Counter
	stopChan chan struct{}
}

func newMetrics() *Metrics {
	mt := &Metrics{
		reg:      prometheus.NewRegistry(),
		viewQps:  make(map[string]*Counter),
		stopChan: make(chan struct{}),
	}

	mt.reg.MustRegister(RequestCount)
	mt.reg.MustRegister(RequestDuration)
	mt.reg.MustRegister(ResponseCount)
	mt.reg.MustRegister(UpdateCount)
	mt.reg.MustRegister(QPS)
	mt.reg.MustRegister(CacheSize)
	mt.reg.MustRegister(CacheHits)

	return mt
}

func Run() {
	timer := time.NewTicker(1 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-gMetrics.stopChan:
			gMetrics.stopChan <- struct{}{}
			return
		case <-timer.C:
		}
		for view, counter := range gMetrics.viewQps {
			QPS.WithLabelValues("server", view).Set(float64(counter.Count()))
			counter.Clear()
		}
	}
}

func Stop() {
	gMetrics.stopChan <- struct{}{}
	<-gMetrics.stopChan
	gMetrics.viewQps = make(map[string]*Counter)
}

func Handler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		handler := promhttp.HandlerFor(gMetrics.reg, promhttp.HandlerOpts{})
		handler.ServeHTTP(w, r)
	}
}

func RecordMetrics(client core.Client) {
	if client.Request.Header.Opcode == g53.OP_QUERY {
		recordQps(TotalView)
		recordQps(client.View)
		RequestCount.WithLabelValues("server", TotalView).Inc()
		RequestCount.WithLabelValues("server", client.View).Inc()
		if client.Response != nil {
			ResponseCount.WithLabelValues("server", client.View).Inc()
			ResponseCount.WithLabelValues("server", TotalView).Inc()
		}
	} else if client.Request.Header.Opcode == g53.OP_UPDATE {
		if client.Response != nil {
			UpdateCount.WithLabelValues("server", TotalView).Inc()
			if client.View != "" {
				UpdateCount.WithLabelValues("server", client.View).Inc()
			}
		}
	}
}

func recordQps(view string) {
	var counter *Counter
	var ok bool
	if counter, ok = gMetrics.viewQps[view]; ok == false {
		counter = newCounter()
		gMetrics.viewQps[view] = counter
	}

	counter.Inc()
}

func RecordCacheHit(view string) {
	CacheHits.WithLabelValues("cache", view).Inc()
	CacheHits.WithLabelValues("cache", TotalView).Inc()
}

func RecordCacheSize(view string, size int, totalSize int) {
	CacheSize.WithLabelValues("cache", view).Set(float64(size))
	CacheSize.WithLabelValues("cache", TotalView).Set(float64(totalSize))
}
