package kubernetes

const (
	// label keys
	LabelPrefix            = "j2."
	LabelJobName           = LabelPrefix + "job.name"
	LabelTaskGroupName     = LabelPrefix + "taskgroup.name"
	LabelTaskGroupFullName = LabelPrefix + "taskgroup.fullname"
	LabelPodName           = LabelPrefix + "pod.name"
)
