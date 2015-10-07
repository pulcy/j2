package main

import (
	"fmt"
	"time"

	"github.com/juju/errgo"
	"github.com/spf13/pflag"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/jobs"
)

func initDeploymentFlags(fs *pflag.FlagSet, f *fg.Flags) {
	fs.StringVarP(&f.JobPath, "job", "j", defaultJobPath, "filename of the job description")
	fs.StringVarP(&f.Stack, "stack", "s", defaultStack, "stack name of the cluster")
	fs.StringVar(&f.Domain, "domain", defaultDomain, "domain name of the cluster")
	fs.StringVarP(&f.Tunnel, "tunnel", "t", defaultTunnel, "SSH endpoint to tunnel through with fleet")
	fs.StringSliceVarP(&f.Groups, "groups", "g", defaultGroups, "target task groups to deploy")
	fs.BoolVarP(&f.Force, "force", "f", defaultForce, "wheather to confirm destroy or not")
	fs.BoolVarP(&f.DryRun, "dry-run", "d", defaultDryRun, "wheather to schedule units or not")
	fs.UintVar(&f.ScalingGroup, "scaling-group", defaultScalingGroup, "scaling group to deploy")
	fs.StringVar(&f.PrivateRegistry, "private-registry", defaultPrivateRegistry, "private registry for the docker images")
	fs.StringVar(&f.LogLevel, "log-level", defaultLogLevel, "log-level for our services")
	fs.BoolVarP(&f.Local, "local", "l", defaultLocal, "User local vagrant test cluster")
	fs.DurationVar(&f.StopDelay, "stop-delay", defaultStopDelay, "Time between stop and destroy")
	fs.DurationVar(&f.DestroyDelay, "destroy-delay", defaultDestroyDelay, "Time between destroy and re-create")
	fs.DurationVar(&f.SliceDelay, "slice-delay", defaultSliceDelay, "Time between update of scaling slices")
}

func deploymentDefaults(f *fg.Flags, args []string) {
	if f.Local {
		f.Tunnel = "core-01"
		f.Stack = "core-01"
		f.StopDelay = 5 * time.Second
		f.DestroyDelay = 3 * time.Second
		f.SliceDelay = 5 * time.Second
	}
	if f.Tunnel == "" {
		f.Tunnel = fmt.Sprintf("%s.%s", f.Stack, f.Domain)
	}

	if f.LogLevel == "" {
		f.LogLevel = "debug"
	}

	if f.JobPath == "" && len(args) == 1 {
		f.JobPath = args[0]
	}
}

func deploymentValidators(f *fg.Flags) {
	if f.Stack == "" || f.Tunnel == "" {
		Exitf("--stack or --tunnel missing")
	}
}

// Gets the list of group names to operate on based on the deployment flags.
func groups(f *fg.Flags) []jobs.TaskGroupName {
	names := []jobs.TaskGroupName{}
	for _, n := range f.Groups {
		names = append(names, jobs.TaskGroupName(n))
	}
	return names
}

// loadJob loads the a job from the given flags.
func loadJob(f *fg.Flags) (*jobs.Job, error) {
	if f.JobPath == "" {
		return nil, maskAny(errgo.New("--job missing"))
	}
	job, err := jobs.ParseJobFromFile(f.JobPath)
	if err != nil {
		return nil, maskAny(err)
	}
	return job, nil
}
