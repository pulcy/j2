package kubernetes

import (
	"strconv"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
)

const (
	prometheusScrapeAnnotation = "prometheus.io/scrape"
	prometheusSchemeAnnotation = "prometheus.io/scheme"
	prometheusPathAnnotation   = "prometheus.io/path"
	prometheusPortAnnotation   = "prometheus.io/port"
)

// addMetricsAnnotations adds all annotations needed to honor the metrics configs in the given pod.
func addMetricsAnnotations(meta *k8s.ObjectMeta, pod pod) {
	for _, t := range pod.tasks {
		metrics := t.Metrics
		if metrics != nil {
			addPrometheusAnnotations(meta, metrics)
			return
		}
	}
	addPrometheusAnnotations(meta, nil)
}

// addPrometheusAnnotations adds prometheus specific annotations.
func addPrometheusAnnotations(meta *k8s.ObjectMeta, metrics *jobs.Metrics) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	if metrics == nil {
		meta.Annotations[prometheusScrapeAnnotation] = "false"
	} else {
		meta.Annotations[prometheusScrapeAnnotation] = "true"
		meta.Annotations[prometheusPathAnnotation] = metrics.Path
		meta.Annotations[prometheusPortAnnotation] = strconv.Itoa(metrics.Port)
	}
}
