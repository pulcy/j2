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

package fleet

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/pulcy/prometheus-conf-api"

	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/sdunits"
)

// addMetricsRegistration adds registration code for metrics to the given units
func addMetricsRegistration(t *jobs.Task, main *sdunits.Unit, ctx generatorContext) error {
	if t.Metrics == nil {
		return nil
	}
	serviceName := t.ServiceName()
	targetServiceName := serviceName
	if t.Type == "proxy" {
		targetServiceName = t.Target.EtcdServiceName()
	}
	key := fmt.Sprintf("/pulcy/metrics/%s-%d", serviceName, ctx.ScalingGroup)
	record := api.MetricsServiceRecord{
		ServiceName: targetServiceName,
		ServicePort: t.Metrics.Port,
		MetricsPath: t.Metrics.Path,
		RulesPath:   t.Metrics.RulesPath,
	}

	json, err := json.Marshal(&record)
	if err != nil {
		return maskAny(err)
	}
	main.ProjectSetting("MetricsRegistration", key+"="+string(json))
	main.ExecOptions.ExecStartPost = append(main.ExecOptions.ExecStartPost,
		fmt.Sprintf("/bin/sh -c 'echo %s | base64 -d | /usr/bin/etcdctl set %s'", base64.StdEncoding.EncodeToString(json), key),
	)
	main.ExecOptions.ExecStop = append(
		[]string{fmt.Sprintf("-/usr/bin/etcdctl rm %s", key)},
		main.ExecOptions.ExecStop...,
	)
	return nil
}
