package kubernetes

import (
	"github.com/pulcy/j2/jobs"

	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

// createDeployments creates all deployments needed for the given task group.
func createDeployments(tg *jobs.TaskGroup, ctx generatorContext) ([]v1beta1.Deployment, error) {
	if tg.Global {
		// Global is mapped onto DaemonSets.
		return nil, nil
	}

	// TODO
	return nil, nil
}

// createDaemonSets creates all daemon sets needed for the given task group.
func createDaemonSets(tg *jobs.TaskGroup, ctx generatorContext) ([]v1beta1.DaemonSet, error) {
	if !tg.Global {
		// Non-global is mapped onto Deployments.
		return nil, nil
	}

	// TODO
	return nil, nil
}
