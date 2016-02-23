// Copyright (c) 2016 Pulcy.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package units

import (
	"fmt"
	"sort"
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
	for _, x := range distinct(u.ExecOptions.wants) {
		lines = append(lines, "Wants="+x)
	}
	for _, x := range distinct(u.ExecOptions.Requires) {
		lines = append(lines, "Requires="+x)
	}
	for _, x := range distinct(u.ExecOptions.BindsTos) {
		lines = append(lines, "BindsTo="+x)
	}
	for _, x := range distinct(u.ExecOptions.after) {
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
		envKeys := []string{}
		for k := range u.ExecOptions.Environment {
			envKeys = append(envKeys, k)
		}
		sort.Strings(envKeys)
		for _, k := range envKeys {
			v := u.ExecOptions.Environment[k]
			lines = append(lines, fmt.Sprintf("Environment=%s", strconv.Quote(fmt.Sprintf("%s=%s", k, v))))
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

		for _, x := range u.ExecOptions.ExecStop {
			lines = append(lines, "ExecStop="+x)
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
	for _, x := range distinct(u.FleetOptions.ConflictsWith) {
		lines = append(lines, "Conflicts="+x)
	}
	if u.FleetOptions.MachineOf != "" {
		lines = append(lines, "MachineOf="+u.FleetOptions.MachineOf)
	}
	if u.FleetOptions.MachineID != "" {
		lines = append(lines, "MachineID="+u.FleetOptions.MachineID)
	}
	for _, x := range distinct(u.FleetOptions.Metadata) {
		lines = append(lines, "MachineMetadata="+x)
	}
	lines = append(lines, "")

	lines = append(lines,
		fmt.Sprintf("[X-%s]", ctx.ProjectName),
		fmt.Sprintf("GeneratedBy=\"%s %s, build %s\"", ctx.ProjectName, ctx.ProjectVersion, ctx.ProjectBuild),
	)
	projectSettingsLines := []string{}
	for k, v := range u.projectSettings {
		projectSettingsLines = append(projectSettingsLines, fmt.Sprintf("%s=%s", k, strconv.Quote(v)))
	}
	sort.Strings(projectSettingsLines)
	lines = append(lines, projectSettingsLines...)
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}

// distinct returns a list with all distinct values of the given list
func distinct(list []string) []string {
	result := []string{}
	seen := make(map[string]struct{})
	for _, x := range list {
		if _, ok := seen[x]; ok {
			continue
		}
		result = append(result, x)
		seen[x] = struct{}{}
	}
	return result
}
