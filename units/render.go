package units

import (
	"strings"
)

func (u *Unit) Render() string {
	lines := []string{
		"[Unit]",
		"Description=" + u.Description,
	}
	for _, x := range u.ExecOptions.Wants {
		lines = append(lines, "Wants="+x)
	}
	for _, x := range u.ExecOptions.BindsTos {
		lines = append(lines, "BindsTo="+x)
	}
	if u.ExecOptions.After != "" {
		lines = append(lines, "After="+u.ExecOptions.After)
	}
	lines = append(lines, "")

	lines = append(lines, "[Service]")
	if u.ExecOptions.IsOneshot != "" {
		lines = append(lines, "Type=oneshot")
	}
	if u.ExecOptions.RemainsAfterExit != "" {
		lines = append(lines, "RemainAfterExit=yes")
	}
	if !u.ExecOptions.IsOneshot != "" {
		lines = append(lines, "Restart="+u.ExecOptions.Restart)
		lines = append(lines, "RestartSec="+u.ExecOptions.RestartSec)
		lines = append(lines, "StartLimitInterval="+u.ExecOptions.StartLimitInterval)
		lines = append(lines, "StartLimitBurst="+u.ExecOptions.StartLimitBurst)
	}
	lines = append(lines, "TimeoutStartSec="+u.ExecOptions.TimeoutStartSec)
	for _, x := range u.ExecOptions.EnvironmentFiles {
		lines = append(lines, "EnvironmentFile="+x)
	}
	for k, v := range u.ExecOptions.Environment {
		lines = append(lines, fmt.Sprintf("Environment=\"%s=%s\"", k, v))
	}

	for _, x := range u.ExecOptions.ExecStartPre {
		lines = append(lines, "ExecStartPre="+x)
	}
	if u.ExecOptions.ExecStart != "" {
		lines = append(lines, "ExecStart="+u.ExecOptions.ExecStart)
	}
	for _, x := range u.ExecOptions.ExecStartPost {
		lines = append(lines, "ExecStartPost="+x)
	}

	for _, x := range u.ExecOptions.ExecStopPre {
		lines = append(lines, "ExecStopPre="+x)
	}
	if u.ExecOptions.ExecStop != "" {
		lines = append(lines, "ExecStop="+u.ExecOptions.ExecStop)
	}
	for _, x := range u.ExecOptions.ExecStopPost {
		lines = append(lines, "ExecStopPost="+x)
	}
	lines = append(lines, "")

	lines = append(lines, "[X-Fleet]")
	if u.FleetOptions.IsGlobal != "" {
		lines = append(lines, "Global=true")
	}
	for _, x := range u.FleetOptions.ConflictsWith {
		lines = append(lines, "Conflicts="+x)
	}
	if u.FleetOptions.MachineOf != "" {
		lines = append(lines, "MachineOf="+u.FleetOptions.MachineOf)
	}
	for _, x := range u.FleetOptions.Metadata {
		lines = append(lines, "MachineMetadata="+x)
	}

	return strings.Join(lines, "\n")
}
