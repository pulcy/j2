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

package fleet_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/juju/errgo"
	"github.com/kr/pretty"
	"github.com/op/go-logging"

	"github.com/pulcy/j2/cluster"
	"github.com/pulcy/j2/engine/docker"
	fg "github.com/pulcy/j2/flags"
	"github.com/pulcy/j2/jobs"
	"github.com/pulcy/j2/pkg/vault"
	"github.com/pulcy/j2/render"
	"github.com/pulcy/j2/render/fleet"
)

const (
	fixtureDir = "./test-fixtures"
)

var (
	maskAny = errgo.MaskFunc(errgo.Any)
)

func TestParse(t *testing.T) {
	cases := []struct {
		Name                    string
		ErrorExpected           bool
		ExpectedUnitNamesCount1 []string
		ExpectedUnitNamesCount3 []string
	}{
		{
			"simple.hcl",
			false,
			[]string{
				"test-couchdb-couchdb-mn@1.service",
				"test-db-db-mn@1.service",
				"test-dummy-dummy-mn@1.service",
				"test-dummy-dummy-mn@2.service",
				"test-dummy-dummy-mn@3.service",
				"test-global-global-mn.service",
				"test-gogs-gogs-mn@1.service",
				"test-registrator-registrator-mn.service",
				"test-some_domain_proxy-some_domain_proxy-mn@1.service",
				"test-some_proxy-some_proxy-mn@1.service",
				"test-web-backup-mn@1.service",
				"test-web-backup-ti@1.timer",
				"test-web-backup-mn@2.service",
				"test-web-backup-ti@2.timer",
				"test-web-nginx-mn@1.service",
				"test-web-nginx-mn@2.service",
				"test-web-storage-mn@1.service",
				"test-web-storage-mn@2.service",
				"test-web-storage-pr0@1.service",
				"test-web-storage-pr0@2.service",
			},
			[]string{
				"test-couchdb-couchdb-mn@1.service",
				"test-db-db-mn@1.service",
				"test-dummy-dummy-mn@1.service",
				"test-dummy-dummy-mn@2.service",
				"test-dummy-dummy-mn@3.service",
				"test-global-global-mn.service",
				"test-gogs-gogs-mn@1.service",
				"test-registrator-registrator-mn.service",
				"test-some_domain_proxy-some_domain_proxy-mn@1.service",
				"test-some_proxy-some_proxy-mn@1.service",
				"test-web-backup-mn@1.service",
				"test-web-backup-ti@1.timer",
				"test-web-backup-mn@2.service",
				"test-web-backup-ti@2.timer",
				"test-web-nginx-mn@1.service",
				"test-web-nginx-mn@2.service",
				"test-web-storage-mn@1.service",
				"test-web-storage-mn@2.service",
				"test-web-storage-pr0@1.service",
				"test-web-storage-pr0@2.service",
			},
		},
		{
			"restart-all.hcl",
			false,
			[]string{
				"restartall-lb1-ta-mn.service",
				"restartall-lb1-tb-mn.service",
				"restartall-lb2-ta-mn.service",
				"restartall-lb2-tb-mn.service",
			},
			[]string{
				"restartall-lb1-ta-mn.service",
				"restartall-lb1-tb-mn.service",
				"restartall-lb2-ta-mn@1.service",
				"restartall-lb2-ta-mn@2.service",
				"restartall-lb2-tb-mn@1.service",
				"restartall-lb2-tb-mn@2.service",
			},
		},
		{
			"secret.hcl",
			false,
			[]string{
				"secrets-env_secrets-env_secrets-mn@1.service",
			},
			[]string{
				"secrets-env_secrets-env_secrets-mn@1.service",
			},
		},
		{
			"proxy.hcl",
			false,
			[]string{
				"proxy-p1-redis-mn@1.service",
				"proxy-p2-p2-mn@1.service",
				"proxy-p2-p2-pr0@1.service",
			},
			[]string{
				"proxy-p1-redis-mn@1.service",
				"proxy-p2-p2-mn@1.service",
				"proxy-p2-p2-pr0@1.service",
			},
		},
		{
			"extra-fields.hcl",
			true,
			[]string{},
			[]string{},
		},
		{
			"variables.hcl",
			false,
			[]string{
				"test-couchdb-couchdb-mn@1.service",
				"test-db-db-mn@1.service",
				"test-dummy-dummy-mn@1.service",
				"test-dummy-dummy-mn@2.service",
				"test-dummy-dummy-mn@3.service",
				"test-global-global-mn.service",
				"test-registrator-registrator-mn.service",
				"test-web-backup-mn@1.service",
				"test-web-backup-ti@1.timer",
				"test-web-backup-mn@2.service",
				"test-web-backup-ti@2.timer",
				"test-web-nginx-mn@1.service",
				"test-web-nginx-mn@2.service",
				"test-web-storage-mn@1.service",
				"test-web-storage-mn@2.service",
				"test-web-storage-pr0@1.service",
				"test-web-storage-pr0@2.service",
			},
			[]string{
				"test-couchdb-couchdb-mn@1.service",
				"test-db-db-mn@1.service",
				"test-dummy-dummy-mn@1.service",
				"test-dummy-dummy-mn@2.service",
				"test-dummy-dummy-mn@3.service",
				"test-global-global-mn.service",
				"test-registrator-registrator-mn.service",
				"test-web-backup-mn@1.service",
				"test-web-backup-ti@1.timer",
				"test-web-backup-mn@2.service",
				"test-web-backup-ti@2.timer",
				"test-web-nginx-mn@1.service",
				"test-web-nginx-mn@2.service",
				"test-web-storage-mn@1.service",
				"test-web-storage-mn@2.service",
				"test-web-storage-pr0@1.service",
				"test-web-storage-pr0@2.service",
			},
		},
		{
			"volumes.hcl",
			false,
			[]string{
				"volumes-v1-backup-mn@1.service",
				"volumes-v1-backup-ti@1.timer",
				"volumes-v1-storage-mn@1.service",
				"volumes-v1-storage-vl0@1.service",
				"volumes-v2-v2-mn@1.service",
				"volumes-v3-v3-mn@1.service",
				"volumes-v3-v3-vl0@1.service",
				"volumes-v4global-v4global-mn.service",
				"volumes-v4global-v4global-vl0.service",
			},
			[]string{
				"volumes-v1-backup-mn@1.service",
				"volumes-v1-backup-ti@1.timer",
				"volumes-v1-storage-mn@1.service",
				"volumes-v1-storage-vl0@1.service",
				"volumes-v2-v2-mn@1.service",
				"volumes-v3-v3-mn@1.service",
				"volumes-v3-v3-vl0@1.service",
				"volumes-v4global-v4global-mn.service",
				"volumes-v4global-v4global-vl0.service",
			},
		},
		{
			"metrics.hcl",
			false,
			[]string{
				"metrics-custom_web-server-mn@1.service",
				"metrics-default_web-server-mn@1.service",
				"metrics-default_web-server_rules-mn@1.service",
				"metrics-web-server-mn@1.service",
				"metrics-web-server-mn@2.service",
			},
			[]string{
				"metrics-custom_web-server-mn@1.service",
				"metrics-default_web-server-mn@1.service",
				"metrics-default_web-server_rules-mn@1.service",
				"metrics-web-server-mn@1.service",
				"metrics-web-server-mn@2.service",
			},
		},
		{
			"constraints.hcl",
			false,
			[]string{
				"constraints-group1-taska-mn@1.service",
				"constraints-group1-taska-mn@2.service",
				"constraints-group2-taskb-mn@1.service",
				"constraints-group2global-taskgrobalb-mn.service",
			},
			[]string{
				"constraints-group1-taska-mn@1.service",
				"constraints-group1-taska-mn@2.service",
				"constraints-group2-taskb-mn@1.service",
				"constraints-group2global-taskgrobalb-mn.service",
			},
		},
		{
			"include.hcl",
			false,
			[]string{
				"include-p1-redis-mn@1.service",
				"include-p2-p2-mn@1.service",
				"include-p2-p2-pr0@1.service",
			},
			[]string{
				"include-p1-redis-mn@1.service",
				"include-p2-p2-mn@1.service",
				"include-p2-p2-pr0@1.service",
			},
		},
	}

	for _, tc := range cases {
		t.Logf("testing %s", tc.Name)
		options := fg.Options{}
		options.Set("option1=value1")
		options.Set("option2=value2")
		cluster3 := cluster.New("test.com", "stack", 3)

		log := logging.MustGetLogger("test")
		vaultConfig := vault.VaultConfig{}
		ghLoginData := vault.GithubLoginData{}
		job, err := jobs.ParseJobFromFile(filepath.Join(fixtureDir, tc.Name), cluster3, options, log, vaultConfig, ghLoginData)
		if tc.ErrorExpected {
			if err == nil {
				t.Fatalf("Expected error in %s", tc.Name)
			}
		} else {
			if err != nil {
				t.Fatalf("Got error in %s: %#v", tc.Name, maskAny(err))
			}
			json, err := job.Json()
			if err != nil {
				t.Fatalf("Cannot convert %s to json: %#v", tc.Name, maskAny(err))
			}
			expectedJson, err := ioutil.ReadFile(filepath.Join(fixtureDir, tc.Name+".json"))
			if err != nil {
				t.Fatalf("Cannot read expected json for %s: %#v", tc.Name, maskAny(err))
			}
			if diffs, err := compareJson(json, expectedJson); err != nil {
				t.Fatalf("Cannot comparse json: %#v", maskAny(err))
			} else if len(diffs) > 0 {
				t.Fatalf("JSON diffs in %s\n%s\nGot: %s", tc.Name, strings.Join(diffs, "\n"), json)
			}

			// Now generate units
			testUnits(t, *job, cluster3, tc.ExpectedUnitNamesCount3, tc.Name)
		}

		cluster1 := cluster.New("test.com", "stack", 1)
		job1, err := jobs.ParseJobFromFile(filepath.Join(fixtureDir, tc.Name), cluster1, options, log, vaultConfig, ghLoginData)
		if tc.ErrorExpected {
			if err == nil {
				t.Fatalf("Expected error in %s", tc.Name)
			}
		} else {
			if err != nil {
				t.Fatalf("Got error in %s: %#v", tc.Name, maskAny(err))
			}

			// Now generate units
			testUnits(t, *job1, cluster1, tc.ExpectedUnitNamesCount1, tc.Name)
		}
	}
}

type renderContext struct {
	projectName    string
	projectVersion string
	projectBuild   string
}

func (r *renderContext) ProjectName() string {
	return r.projectName
}

func (r *renderContext) ProjectVersion() string {
	return r.projectVersion
}

func (r *renderContext) ProjectBuild() string {
	return r.projectBuild
}

func testUnits(t *testing.T, job jobs.Job, cl cluster.Cluster, expectedUnitNames []string, testName string) {
	fleet.FixedPwhashSalt = "test-salt"
	config := render.RenderConfig{
		Groups:              nil,
		CurrentScalingGroup: 0,
		DockerOptions: cluster.DockerOptions{
			LoggingArgs: []string{"--log-driver=test"},
		},
		FleetOptions: cl.FleetOptions,
	}
	generator := fleet.NewGenerator(job, config)
	ctx := &renderContext{
		projectName:    "testproject",
		projectVersion: "test-version",
		projectBuild:   "test-build",
	}
	docker.SetupImages(docker.Images{
		VaultMonkey: "pulcy/vault-monkey:latest",
		Wormhole:    "pulcy/wormhole:latest",
		Alpine:      "alpine:3.3",
		CephVolume:  "pulcy/ceph-volume:latest",
	})
	units, err := generator.GenerateUnits(ctx, cl.InstanceCount)
	if err != nil {
		t.Fatalf("GenerateUnits failed for instance-count %d: %#v", cl.InstanceCount, maskAny(err))
	}
	compareUnitNames(t, expectedUnitNames, units)
	compareUnitFiles(t, units, filepath.Join(fixtureDir, "units", fmt.Sprintf("instance-count-%d", cl.InstanceCount), testName))
}

func compareJson(a, b []byte) ([]string, error) {
	oa := make(map[string]interface{})
	if err := json.Unmarshal(a, &oa); err != nil {
		return nil, maskAny(err)
	}

	ob := make(map[string]interface{})
	if err := json.Unmarshal(b, &ob); err != nil {
		return nil, maskAny(err)
	}

	diffs := pretty.Diff(oa, ob)
	return diffs, nil
}

func compareUnitNames(t *testing.T, expected []string, generated []render.UnitData) {
	var found []string
	for _, u := range generated {
		found = append(found, u.Name())
	}
	sort.Strings(expected)
	sort.Strings(found)
	expectedStr := strings.Join(expected, "\n- ")
	foundStr := strings.Join(found, "\n- ")
	if expectedStr != foundStr {
		t.Fatalf("Unexpected unit names. Expected \n- %s\ngot \n- %s", expectedStr, foundStr)
	}
}

func compareUnitFiles(t *testing.T, units []render.UnitData, fixtureDir string) {
	errors := []string{}
	tmpDir, err := ioutil.TempDir("", "j2-test")
	if err != nil {
		t.Fatalf("Cannot create temp dir: %#v", err)
		return
	}
	defer os.RemoveAll(tmpDir)
	for _, u := range units {
		fixturePath := filepath.Join(fixtureDir, u.Name())
		if _, err := os.Stat(fixturePath); os.IsNotExist(err) || os.Getenv("UPDATE-FIXTURES") == "1" {
			// Fixture does not yet exist, create it
			os.MkdirAll(fixtureDir, 0755)
			if err := ioutil.WriteFile(fixturePath, []byte(u.Content()), 0755); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to create fixture: %#v", maskAny(err)))
			}
		} else {
			// Compare
			fixtureRaw, err := ioutil.ReadFile(fixturePath)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to read fixture: %#v", maskAny(err)))
				continue
			}

			fixtureContent := strings.TrimSpace(string(fixtureRaw))
			fnContent := strings.TrimSpace(u.Content())

			if fixtureContent != fnContent {
				fn := filepath.Join(tmpDir, u.Name())
				if err := ioutil.WriteFile(fn, []byte(u.Content()), 0755); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to create fn: %#v", maskAny(err)))
				} else {
					cmd := exec.Command("diff", fixturePath, fn)
					if output, err := cmd.Output(); err != nil {
						errors = append(errors, fmt.Sprintf("File '%s' is different:\n%s", fixturePath, string(output)))
					}
				}
			}
		}
	}
	if len(errors) > 0 {
		t.Fatal(strings.Join(errors, "\n"))
	}
}
