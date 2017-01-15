package kubernetes

import (
	k8s "github.com/YakLabs/k8s-client"
	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
)

// setLabel sets a label in the given meta data, creating the Labels map when needed
func setLabel(m *k8s.ObjectMeta, key, value string) {
	if m.Labels == nil {
		m.Labels = make(map[string]string)
	}
	m.Labels[key] = value
}

// setAnnotation sets an annotation in the given meta data, creating the Annotations map when needed
func setAnnotation(m *k8s.ObjectMeta, key, value string) {
	if m.Annotations == nil {
		m.Annotations = make(map[string]string)
	}
	m.Annotations[key] = value
}

// setJobLabels adds all job related labels to the given meta.
func setJobLabels(m *k8s.ObjectMeta, j *jobs.Job) {
	setLabel(m, pkg.LabelJobName, pkg.ResourceName(j.Name.String()))
}

// setTaskGroupLabelsAnnotations adds all task group related labels to the given meta.
// This includes all job related labels.
func setTaskGroupLabelsAnnotations(m *k8s.ObjectMeta, tg *jobs.TaskGroup) {
	setJobLabels(m, tg.Job())
	setLabel(m, pkg.LabelTaskGroupName, pkg.ResourceName(tg.Name.String()))
	setLabel(m, pkg.LabelTaskGroupFullName, pkg.ResourceName(tg.FullName()))
}

// setPodLabels adds all pod related labels to the given meta.
// This includes all job & task group related labels.
func setPodLabels(m *k8s.ObjectMeta, tg *jobs.TaskGroup, pod pod) {
	setTaskGroupLabelsAnnotations(m, tg)
	setLabel(m, pkg.LabelPodName, pkg.ResourceName(pod.name))
}

func createPodSelector(input map[string]string, pod pod) map[string]string {
	if input == nil {
		input = make(map[string]string)
	}
	input[pkg.LabelPodName] = pkg.ResourceName(pod.name)
	return input
}
