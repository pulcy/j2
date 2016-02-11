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

package jobs_test

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

	"github.com/kr/pretty"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/jobs"
	"arvika.pulcy.com/pulcy/deployit/units"
)

const (
	fixtureDir = "./test-fixtures"
)

func TestParse(t *testing.T) {
	cases := []struct {
		Name              string
		ErrorExpected     bool
		ExpectedUnitNames []string
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
				"test-registrator-registrator-mn.service",
				"test-web-backup-mn@1.service",
				"test-web-backup-ti@1.timer",
				"test-web-backup-mn@2.service",
				"test-web-backup-ti@2.timer",
				"test-web-nginx-mn@1.service",
				"test-web-nginx-mn@2.service",
				"test-web-storage-mn@1.service",
				"test-web-storage-mn@2.service",
			},
		},
		{
			"secret.hcl",
			false,
			[]string{
				"secrets-env_secrets-env_secrets-sc@1.service",
				"secrets-env_secrets-env_secrets-mn@1.service",
			},
		},
		{
			"extra-fields.hcl",
			true,
			[]string{},
		},
	}

	for _, tc := range cases {
		t.Logf("testing %s", tc.Name)
		options := fg.Options{}
		options.Set("option1=value1")
		options.Set("option2=value2")
		cluster := fg.Cluster{
			Domain:        "test.com",
			Stack:         "stack",
			InstanceCount: 3,
		}
		job, err := jobs.ParseJobFromFile(filepath.Join(fixtureDir, tc.Name), cluster, options)
		if tc.ErrorExpected {
			if err == nil {
				t.Fatalf("Expected error in %s", tc.Name)
			}
		} else {
			if err != nil {
				t.Fatalf("Got error in %s: %#v", tc.Name, err)
			}
			json, err := job.Json()
			if err != nil {
				t.Fatalf("Cannot convert %s to json: %#v", tc.Name, err)
			}
			expectedJson, err := ioutil.ReadFile(filepath.Join(fixtureDir, tc.Name+".json"))
			if err != nil {
				t.Fatalf("Cannot read expected json for %s: %#v", tc.Name, err)
			}
			if diffs, err := compareJson(json, expectedJson); err != nil {
				t.Fatalf("Cannot comparse json: %#v", err)
			} else if len(diffs) > 0 {
				t.Fatalf("JSON diffs in %s\n%s\nGot: %s", tc.Name, strings.Join(diffs, "\n"), json)
			}

			// Now generate units
			testUnits(t, job, 1, tc.ExpectedUnitNames, tc.Name)
			testUnits(t, job, 3, tc.ExpectedUnitNames, tc.Name)
		}
	}
}

func testUnits(t *testing.T, job *jobs.Job, instanceCount int, expectedUnitNames []string, testName string) {
	jobs.FixedPwhashSalt = "test-salt"
	generator := job.Generate(nil, 0)
	ctx := units.RenderContext{
		ProjectName:    "testproject",
		ProjectVersion: "test-version",
		ProjectBuild:   "test-build",
	}
	defer generator.RemoveTmpFiles()
	images := jobs.Images{
		VaultMonkey: "pulcy/vault-monkey:latest",
	}
	if err := generator.WriteTmpFiles(ctx, images, instanceCount); err != nil {
		t.Fatalf("WriteTmpFiles failed: %#v", err)
	}
	compareUnitNames(t, expectedUnitNames, generator.UnitNames())
	compareUnitFiles(t, generator.FileNames(), filepath.Join(fixtureDir, "units", fmt.Sprintf("instance-count-%d", instanceCount), testName))
}

func compareJson(a, b []byte) ([]string, error) {
	oa := make(map[string]interface{})
	if err := json.Unmarshal(a, &oa); err != nil {
		return nil, err
	}

	ob := make(map[string]interface{})
	if err := json.Unmarshal(b, &ob); err != nil {
		return nil, err
	}

	diffs := pretty.Diff(oa, ob)
	return diffs, nil
}

func compareUnitNames(t *testing.T, expected, found []string) {
	sort.Strings(expected)
	sort.Strings(found)
	expectedStr := strings.Join(expected, ",")
	foundStr := strings.Join(found, ",")
	if expectedStr != foundStr {
		t.Fatalf("Unexpected unit names. Expected '%s', got '%s'", expectedStr, foundStr)
	}
}

func compareUnitFiles(t *testing.T, fileNames []string, fixtureDir string) {
	errors := []string{}
	for _, fn := range fileNames {
		fixturePath := filepath.Join(fixtureDir, filepath.Base(fn))
		if _, err := os.Stat(fixturePath); os.IsNotExist(err) || os.Getenv("UPDATE-FIXTURES") == "1" {
			// Fixture does not yet exist, create it
			os.MkdirAll(fixtureDir, 0755)
			data, err := ioutil.ReadFile(fn)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Failed to read '%s': %#v", fn, err))
			} else {
				if err := ioutil.WriteFile(fixturePath, data, 0755); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to create fixture: %#v", err))
				}
			}
		} else {
			// Compare
			cmd := exec.Command("diff", fn, fixturePath)
			if output, err := cmd.Output(); err != nil {
				errors = append(errors, fmt.Sprintf("File '%s' is different:\n%s", fixturePath, string(output)))
			}
		}
	}
	if len(errors) > 0 {
		t.Fatal(strings.Join(errors, "\n"))
	}
}
