package jobs_test

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kr/pretty"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
	"arvika.pulcy.com/pulcy/deployit/jobs"
)

const (
	fixtureDir = "./test-fixtures"
)

func TestParse(t *testing.T) {
	cases := []struct {
		Name          string
		ErrorExpected bool
	}{
		{
			"simple.hcl",
			false,
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
		}
	}
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
