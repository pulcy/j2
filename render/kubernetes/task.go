package kubernetes

import (
	"fmt"

	"github.com/pulcy/j2/jobs"
	"k8s.io/client-go/pkg/api/v1"
)

func createTaskContainers(t *jobs.Task, ctx generatorContext) ([]v1.Container, error) {
	if t.Type.IsService() {
		return createServiceTaskContainers(t, ctx)
	} else if t.Type.IsProxy() {
		return nil, maskAny(fmt.Errorf("Proxy not yet implemented", t.Type))
	} else if t.Type.IsOneshot() {
		return createServiceTaskContainers(t, ctx)
	} else {
		return nil, maskAny(fmt.Errorf("Unknown task type '%s'", t.Type))
	}
}

func createServiceTaskContainers(t *jobs.Task, ctx generatorContext) ([]v1.Container, error) {
	c := v1.Container{
		Name:  t.ContainerName(ctx.ScalingGroup),
		Image: t.Image.String(),
		Args:  t.Args,
	}

	for _, p := range t.Ports {
		cp, err := createContainerPort(p)
		if err != nil {
			return nil, maskAny(err)
		}
		c.Ports = append(c.Ports, cp)
	}

	return []v1.Container{
		c,
	}, nil
}
