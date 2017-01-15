package kubernetes

import (
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/extpoints"
	"github.com/pulcy/j2/jobs"
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
func (o *k8sOrchestrator) Scheduler(j jobs.Job, cluster cluster.Cluster) (scheduler.Scheduler, error) {
	return sk.NewScheduler(j, cluster.KubernetesOptions.KubeConfig, cluster.KubernetesOptions.Context)
}

func init() {
	extpoints.Orchestrators.Register(&k8sOrchestrator{}, "kubernetes")
}
