package kubernetes

import (
	"github.com/pulcy/j2/jobs"
	"k8s.io/client-go/pkg/api/v1"
)

const (
	// metadata keys
	metaPrefix            = "j2."
	metaJobName           = metaPrefix + "job.name"
	metaTaskGroupFullName = metaPrefix + "taskgroup.fullname"
	metaPodName           = metaPrefix + "pod.name"
)

// setLabel sets a label in the given meta data, creating the Labels map when needed
func setLabel(m *v1.ObjectMeta, key, value string) {
	if m.Labels == nil {
		m.Labels = make(map[string]string)
	}
	m.Labels[key] = value
}

// setAnnotation sets an annotation in the given meta data, creating the Annotations map when needed
func setAnnotation(m *v1.ObjectMeta, key, value string) {
	if m.Annotations == nil {
		m.Annotations = make(map[string]string)
	}
	m.Annotations[key] = value
}

// setJobLabels adds all job related labels to the given meta.
func setJobLabels(m *v1.ObjectMeta, j *jobs.Job) {
	setLabel(m, metaJobName, j.Name.String())
}

// setTaskGroupLabelsAnnotations adds all task group related labels to the given meta.
// This includes all job related labels.
func setTaskGroupLabelsAnnotations(m *v1.ObjectMeta, tg *jobs.TaskGroup) {
	setJobLabels(m, tg.Job())
	setLabel(m, metaTaskGroupFullName, tg.FullName())
}

// setPodLabels adds all pod related labels to the given meta.
// This includes all job & task group related labels.
func setPodLabels(m *v1.ObjectMeta, tg *jobs.TaskGroup, pod pod) {
	setTaskGroupLabelsAnnotations(m, tg)
	setLabel(m, metaPodName, pod.name)
}
