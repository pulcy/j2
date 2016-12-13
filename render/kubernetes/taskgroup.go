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

	d := v1beta1.Deployment{}
	d.TypeMeta.Kind = "Deployment"
	d.TypeMeta.APIVersion = "extensions/v1beta1"

	d.ObjectMeta.Name = resourceName(tg.FullName(), kindDeployment)

	count := int32(tg.Count)
	d.Spec.Replicas = &count

	d.Spec.Template.ObjectMeta.SetAnnotations(map[string]string{
		"taskgroup.name": tg.Name.String(),
	})

	for _, t := range tg.Tasks {
		containers, err := createTaskContainers(t, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, containers...)
	}

	return []v1beta1.Deployment{d}, nil
}

// createDaemonSets creates all daemon sets needed for the given task group.
func createDaemonSets(tg *jobs.TaskGroup, ctx generatorContext) ([]v1beta1.DaemonSet, error) {
	if !tg.Global {
		// Non-global is mapped onto Deployments.
		return nil, nil
	}

	d := v1beta1.DaemonSet{}
	d.TypeMeta.Kind = "DaemonSet"
	d.TypeMeta.APIVersion = "extensions/v1beta1"

	d.ObjectMeta.Name = resourceName(tg.FullName(), kindDaemonSet)

	d.Spec.Template.ObjectMeta.SetAnnotations(map[string]string{
		"taskgroup.name": tg.Name.String(),
	})

	for _, t := range tg.Tasks {
		containers, err := createTaskContainers(t, ctx)
		if err != nil {
			return nil, maskAny(err)
		}
		d.Spec.Template.Spec.Containers = append(d.Spec.Template.Spec.Containers, containers...)
	}

	return []v1beta1.DaemonSet{d}, nil
}
