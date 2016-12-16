package kubernetes

import (
	"github.com/ericchiang/k8s"
	"github.com/ericchiang/k8s/api/v1"
	v1beta1 "github.com/ericchiang/k8s/apis/extensions/v1beta1"
	"github.com/pulcy/j2/jobs"
)

// createDeployments creates all deployments needed for the given task group.
func createDeployments(tg *jobs.TaskGroup, pod pod, ctx generatorContext) ([]v1beta1.Deployment, error) {
	if tg.Global {
		// Global is mapped onto DaemonSets.
		return nil, nil
	}

	d := v1beta1.Deployment{
		Metadata: &v1.ObjectMeta{
			Name:      k8s.StringP(resourceName(pod.name, kindDeployment)),
			Namespace: k8s.StringP(ctx.Namespace),
		},
		Spec: &v1beta1.DeploymentSpec{
			Replicas: k8s.Int32P(int32(tg.Count)),
		},
	}
	setTaskGroupLabelsAnnotations(d.GetMetadata(), tg)

	template, err := createPodTemplateSpec(tg, pod, ctx)
	if err != nil {
		return nil, maskAny(err)
	}
	d.Spec.Template = template

	return []v1beta1.Deployment{d}, nil
}
