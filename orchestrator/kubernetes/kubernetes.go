package kubernetes

import (
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/extpoints"
	"github.com/pulcy/j2/render"
	rk "github.com/pulcy/j2/render/kubernetes"
	"github.com/pulcy/j2/scheduler"
	sk "github.com/pulcy/j2/scheduler/kubernetes"
)

type k8sOrchestrator struct{}

// RenderProvider returns the provider for the unit renderer for this orchestrator.
func (o *k8sOrchestrator) RenderProvider() (render.RenderProvider, error) {
	return rk.NewRenderProvider(), nil
}

// Scheduler returns the scheduler, configured for the given cluster, for this orchestrator.
func (o *k8sOrchestrator) Scheduler(cluster cluster.Cluster) (scheduler.Scheduler, error) {
	return sk.NewScheduler(cluster.KubernetesOptions.KubeConfig, cluster.KubernetesOptions.Namespace)
}

func init() {
	extpoints.Orchestrators.Register(&k8sOrchestrator{}, "kubernetes")
}
