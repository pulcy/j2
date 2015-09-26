package services

import (
	"fmt"

	"arvika.pulcy.com/iggi/deployit/units"
)

type DockerService struct {
	name        string
	description string
	registry    string
	image       string
	version     string
	scale       uint8
	global      bool
	args        []string
	ports       []*DockerPort
}

type DockerPort struct {
	hostIP        string
	hostPort      string
	containerPort string
}

func NewPort(containerPort string) *DockerPort {
	return &DockerPort{
		hostIP:        "${COREOS_PRIVATE_IPV4}",
		hostPort:      "",
		containerPort: containerPort,
	}
}

func (dp *DockerPort) HostIP(hostIP string) *DockerPort {
	dp.hostIP = hostIP
	return dp
}

func (dp *DockerPort) HostPort(hostPort string) *DockerPort {
	dp.hostPort = hostPort
	return dp
}

func (dp *DockerPort) String() string {
	return fmt.Sprintf("%s:%s:%s", dp.hostIP, dp.hostPort, dp.containerPort)
}

func NewDockerService(name, description string) *DockerService {
	return &DockerService{
		name:        name,
		description: description,
	}
}

// Image sets docker image info
func (ds *DockerService) Image(registry, image, version string) *DockerService {
	ds.registry = registry
	ds.image = image
	ds.version = version
	return ds
}

// Append arguments
func (ds *DockerService) Args(args ...string) *DockerService {
	ds.args = append(ds.args, args...)
	return ds
}

// Append exposed ports
func (ds *DockerService) Ports(ports ...*DockerPort) *DockerService {
	ds.ports = append(ds.ports, ports...)
	return ds
}

// Scale sets the number of instances of this service to generate
func (ds *DockerService) Scale(scale uint8) *DockerService {
	ds.scale = scale
	return ds
}

// Global marks this service as global (scheduled on all machines)
func (ds *DockerService) Global() *DockerService {
	ds.global = true
	ds.scale = 1
	return ds
}

func (ds *DockerService) Name() string {
	return ds.name
}

// Units generates all units for the given service.
func (ds *DockerService) Units(currentScaleGroup uint8) ([]units.Unit, error) {
	list := []units.Unit{}

	for i := uint8(1); i <= ds.scale; i++ {
		if currentScaleGroup != 0 && currentScaleGroup != i {
			continue
		}
		list = append(list, ds.createMainUnit(i))
	}
	return list, nil
}

func (ds *DockerService) createMainUnit(currentScaleGroup uint8) units.Unit {
	execStart := []string{
		"/usr/bin/docker",
		"run",
		"--rm",
		"--name $NAME",
	}
	for _, p := range ds.ports {
		execStart = append(execStart, "-p "+p.String())
	}
	execStart = append(execStart, "$IMAGE")
	execStart = append(execStart, ds.args...)
	main := units.Unit{
		Name:         ds.name,
		Description:  ds.description,
		Type:         "service",
		Scalable:     ds.scale > 1,
		ScalingGroup: currentScaleGroup,
		ExecOptions:  units.NewExecOptions(execStart...),
		FleetOptions: units.NewFleetOptions(),
	}
	main.FleetOptions.IsGlobal = ds.global
	main.ExecOptions.Environment = map[string]string{
		"NAME":  main.ContainerName(),
		"IMAGE": fmt.Sprintf("%s/%s:%v", ds.registry, ds.image, ds.version),
	}

	return main
}
