package units

import (
	"fmt"
	"strconv"
	"strings"
)

type RenderContext struct {
	ProjectName    string
	ProjectVersion string
	ProjectBuild   string
}

func (u *Unit) Render(ctx RenderContext) string {
	lines := []string{
		"[Unit]",
		"Description=" + u.Description,
	}
	if u.ExecOptions.Wants != "" {
		lines = append(lines, "Wants="+u.ExecOptions.Wants)
	}
	for _, x := range u.ExecOptions.Requires {
		lines = append(lines, "Requires="+x)
	}
	for _, x := range u.ExecOptions.BindsTos {
		lines = append(lines, "BindsTo="+x)
	}
	for _, x := range u.ExecOptions.after {
		lines = append(lines, "After="+x)
	}
	lines = append(lines, "")

	if u.Type == "service" {
		lines = append(lines, "[Service]")
		if u.ExecOptions.IsOneshot {
			lines = append(lines, "Type=oneshot")
		}
		if u.ExecOptions.RemainsAfterExit {
			lines = append(lines, "RemainAfterExit=yes")
		}
		if !u.ExecOptions.IsOneshot {
			lines = append(lines, "Restart="+u.ExecOptions.Restart)
			lines = append(lines, "RestartSec="+strconv.Itoa(int(u.ExecOptions.RestartSec)))
			lines = append(lines, "StartLimitInterval="+u.ExecOptions.StartLimitInterval)
			lines = append(lines, "StartLimitBurst="+strconv.Itoa(int(u.ExecOptions.StartLimitBurst)))
		}
		lines = append(lines, "TimeoutStartSec="+strconv.Itoa(int(u.ExecOptions.TimeoutStartSec)))
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
	} else if u.Type == "timer" {
		lines = append(lines, "[Timer]")
		if u.ExecOptions.OnCalendar != "" {
			lines = append(lines, "OnCalendar="+u.ExecOptions.OnCalendar)
		}
		if u.ExecOptions.Unit != "" {
			lines = append(lines, "Unit="+u.ExecOptions.Unit)
		}
		lines = append(lines, "")
	}

	lines = append(lines, "[X-Fleet]")
	if u.FleetOptions.IsGlobal {
		lines = append(lines, "Global=true")
	}
	for _, x := range u.FleetOptions.ConflictsWith {
		lines = append(lines, "Conflicts="+x)
	}
	if u.ExecOptions.MachineOf != "" {
		lines = append(lines, "MachineOf="+u.ExecOptions.MachineOf)
	}
	for _, x := range u.FleetOptions.Metadata {
		lines = append(lines, "MachineMetadata="+x)
	}
	lines = append(lines, "")

	lines = append(lines,
		fmt.Sprintf("[X-%s]", ctx.ProjectName),
		fmt.Sprintf("GeneratedBy=\"%s %s, build %s\"", ctx.ProjectName, ctx.ProjectVersion, ctx.ProjectBuild),
	)
	for k, v := range u.projectSettings {
		lines = append(lines, fmt.Sprintf("%s=%s", k, strconv.Quote(v)))
	}
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}
