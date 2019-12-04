package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/core"
)

const (
	DefaultView = "default"
	TotalView   = "any"
)

var gMetrics *Metrics

type Metrics struct {
	reg      *prometheus.Registry
	viewQps  map[string]*Counter
	stopChan chan struct{}
}

func NewMetrics(conf *config.VanguardConf) *Metrics {
	gMetrics = &Metrics{
		reg:      prometheus.NewRegistry(),
		stopChan: make(chan struct{}),
	}

	gMetrics.reg.MustRegister(RequestCount)
	gMetrics.reg.MustRegister(ResponseCount)
	gMetrics.reg.MustRegister(UpdateCount)
	gMetrics.reg.MustRegister(QPS)
	gMetrics.reg.MustRegister(CacheSize)
	gMetrics.reg.MustRegister(CacheHits)

	gMetrics.reg.MustRegister(RequestCountByView)
	gMetrics.reg.MustRegister(ResponseCountByView)
	gMetrics.reg.MustRegister(UpdateCountByView)
	gMetrics.reg.MustRegister(QPSByView)
	gMetrics.reg.MustRegister(CacheSizeByView)
	gMetrics.reg.MustRegister(CacheHitsByView)

	gMetrics.ReloadConfig(conf)
	return gMetrics
}

func GetMetrics() *Metrics {
	return gMetrics
}

func (m *Metrics) ReloadConfig(conf *config.VanguardConf) {
	m.viewQps = make(map[string]*Counter)
	m.viewQps[DefaultView] = newCounter()
	m.viewQps[TotalView] = newCounter()
	for _, viewAcl := range conf.Views.ViewAcls {
		m.viewQps[viewAcl.View] = newCounter()
	}
}

func (m *Metrics) Run() {
	timer := time.NewTicker(1 * time.Second)
	defer timer.Stop()

	for {
		select {
		case <-m.stopChan:
			m.stopChan <- struct{}{}
			return
		case <-timer.C:
		}
		for view, counter := range m.viewQps {
			if view == TotalView {
				QPS.WithLabelValues("server").Set(float64(counter.Count()))
			} else {
				QPSByView.WithLabelValues("server", view).Set(float64(counter.Count()))
			}
			counter.Clear()
		}
	}
}

func (m *Metrics) Stop() {
	m.stopChan <- struct{}{}
	<-m.stopChan
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
		RequestCount.WithLabelValues("server").Inc()
		RequestCountByView.WithLabelValues("server", client.View).Inc()
		if client.Response != nil {
			ResponseCount.WithLabelValues("server").Inc()
			ResponseCountByView.WithLabelValues("server", client.View).Inc()
		}
	} else if client.Request.Header.Opcode == g53.OP_UPDATE {
		if client.Response != nil {
			UpdateCount.WithLabelValues("server").Inc()
			if client.View != "" {
				UpdateCountByView.WithLabelValues("server", client.View).Inc()
			}
		}
	}
}

func recordQps(view string) {
	if counter, ok := gMetrics.viewQps[view]; ok {
		counter.Inc()
	}
}

func RecordCacheHit(view string) {
	CacheHits.WithLabelValues("cache").Inc()
	CacheHitsByView.WithLabelValues("cache", view).Inc()
}

func RecordCacheSize(view string, size int, totalSize int) {
	CacheSize.WithLabelValues("cache").Set(float64(totalSize))
	CacheSizeByView.WithLabelValues("cache", view).Set(float64(size))
}
