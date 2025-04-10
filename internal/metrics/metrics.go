package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	pvzCreatedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "pvz_created_total",
		Help: "Total number of PVZ created",
	})
	receptionCreatedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "reception_created_total",
		Help: "Total number of receptions created",
	})
	productAddedCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "product_added_total",
		Help: "Total number of products added",
	})
)

// InitMetrics регистрирует метрики
func InitMetrics() {
	prometheus.MustRegister(pvzCreatedCounter, receptionCreatedCounter, productAddedCounter)
}

func IncPVZCreated() {
	pvzCreatedCounter.Inc()
}

func IncReceptionCreated() {
	receptionCreatedCounter.Inc()
}

func IncProductAdded() {
	productAddedCounter.Inc()
}
