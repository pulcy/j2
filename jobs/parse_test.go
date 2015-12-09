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
				"test-couchdb-couchdb@1.service",
				"test-db-db@1.service",
				"test-dummy-dummy@1.service",
				"test-dummy-dummy@2.service",
				"test-dummy-dummy@3.service",
				"test-global-global.service",
				"test-registrator-registrator.service",
				"test-web-backup@1.service",
				"test-web-backup@2.service",
				"test-web-nginx@1.service",
				"test-web-nginx@2.service",
				"test-web-storage@1.service",
				"test-web-storage@2.service",
			},
		},
	}

	for _, tc := range cases {
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
	generator := job.Generate(nil, 0)
	ctx := units.RenderContext{
		ProjectName:    "testproject",
		ProjectVersion: "test-version",
		ProjectBuild:   "test-build",
	}
	defer generator.RemoveTmpFiles()
	if err := generator.WriteTmpFiles(ctx, instanceCount); err != nil {
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
	for _, fn := range fileNames {
		fixturePath := filepath.Join(fixtureDir, filepath.Base(fn))
		if _, err := os.Stat(fixturePath); os.IsNotExist(err) || os.Getenv("UPDATE-FIXTURES") == "1" {
			// Fixture does not yet exist, create it
			os.MkdirAll(fixtureDir, 0755)
			data, err := ioutil.ReadFile(fn)
			if err != nil {
				t.Fatalf("Failed to read '%s': %#v", fn, err)
			}
			if err := ioutil.WriteFile(fixturePath, data, 0755); err != nil {
				t.Fatalf("Failed to create fixture: %#v", err)
			}
		} else {
			// Compare
			cmd := exec.Command("diff", fn, fixturePath)
			if output, err := cmd.Output(); err != nil {
				t.Fatalf("File '%s' is different:\n%s", fixturePath, string(output))
			}
		}
	}
}
