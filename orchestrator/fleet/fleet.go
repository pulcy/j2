package fleet

import (
	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/extpoints"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/render"
	rf "github.com/pulcy/j2/render/fleet"
	"github.com/pulcy/j2/scheduler"
	sf "github.com/pulcy/j2/scheduler/fleet"
)

type fleetOrchestrator struct{}

// RenderProvider returns the provider for the unit renderer for this orchestrator.
func (o *fleetOrchestrator) RenderProvider() (render.RenderProvider, error) {
	return rf.NewRenderProvider(), nil
}

// Scheduler returns the scheduler, configured for the given cluster, for this orchestrator.
func (o *fleetOrchestrator) Scheduler(j jobs.Job, cluster cluster.Cluster) (scheduler.Scheduler, error) {
	return sf.NewScheduler(cluster.Tunnel)
}

func init() {
	extpoints.Orchestrators.Register(&fleetOrchestrator{}, "fleet")
}
