package kubernetes

import (
	"encoding/json"
	"strconv"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
	api "github.com/pulcy/prometheus-conf-api"
)

const (
	prometheusScrapeAnnotation = "prometheus.io/scrape"
	prometheusSchemeAnnotation = "prometheus.io/scheme"
	prometheusPathAnnotation   = "prometheus.io/path"
	prometheusPortAnnotation   = "prometheus.io/port"
	j2MetricsAnnotation        = "j2.pulcy.com/metrics"
)

// addMetricsAnnotations adds all annotations needed to honor the metrics configs in the given pod.
func addMetricsAnnotations(meta *k8s.ObjectMeta, pod pod) error {
	for _, t := range pod.tasks {
		metrics := t.Metrics
		if metrics != nil {
			if err := addPrometheusAnnotations(meta, metrics); err != nil {
				return maskAny(err)
			}
			if err := addJ2MetricsAnnotations(meta, metrics); err != nil {
				return maskAny(err)
			}
			return nil
		}
	}
	if err := addPrometheusAnnotations(meta, nil); err != nil {
		return maskAny(err)
	}
	return nil
}

// addPrometheusAnnotations adds prometheus specific annotations.
func addPrometheusAnnotations(meta *k8s.ObjectMeta, metrics *jobs.Metrics) error {
	if metrics == nil {
		return nil
	}
	if metrics.Path != "" && metrics.Port != 0 {
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}
		meta.Annotations[prometheusScrapeAnnotation] = "true"
		meta.Annotations[prometheusPathAnnotation] = metrics.Path
		meta.Annotations[prometheusPortAnnotation] = strconv.Itoa(metrics.Port)
	}
	return nil
}

// addJ2MetricsAnnotations adds J2 specific annotations.
func addJ2MetricsAnnotations(meta *k8s.ObjectMeta, metrics *jobs.Metrics) error {
	if metrics == nil {
		return nil
	}
	if metrics.RulesPath != "" {
		if meta.Annotations == nil {
			meta.Annotations = make(map[string]string)
		}
		records := []api.MetricsServiceRecord{
			api.MetricsServiceRecord{
				RulesPath:   metrics.RulesPath,
				ServicePort: metrics.Port,
			},
		}
		if ann, err := json.Marshal(records); err != nil {
			return maskAny(err)
		} else {
			meta.Annotations[j2MetricsAnnotation] = string(ann)
		}
	}
	return nil
}
