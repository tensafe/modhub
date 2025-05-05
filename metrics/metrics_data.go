package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

var (
	model_metric_status_metric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tsdatax_task_status_metric",
			Help: "get tsdatax task status metric",
		},
		[]string{"taskid", "variable_name", "variable_unit"},
	)
	ModelMetricRegistry = prometheus.NewRegistry()
)

func init() {
	ModelMetricRegistry.MustRegister(model_metric_status_metric)
	ModelMetricRegistry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
}
