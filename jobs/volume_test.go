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
	"reflect"
	"testing"

	"github.com/pulcy/j2/jobs"
)

func TestParseVolume(t *testing.T) {
	tests := []struct {
		Input         string
		ErrorExpected bool
		Expected      jobs.Volume
	}{
		{Input: "/tmp:/data", Expected: jobs.Volume{Path: "/data", HostPath: "/tmp", Type: "local"}},
		{Input: "/tmp:/data:ro", Expected: jobs.Volume{Path: "/data", HostPath: "/tmp", Type: "local", Options: []string{"ro"}}},
		{Input: "/var/lib:/var/lib:ro,shared", Expected: jobs.Volume{Path: "/var/lib", HostPath: "/var/lib", Type: "local", Options: []string{"ro", "shared"}}},
		{Input: "/tmp:/data:wierd", ErrorExpected: true},
		{Input: "/foo", Expected: jobs.Volume{Path: "/foo", Type: "instance"}},
		{Input: "instance:/foo/2", Expected: jobs.Volume{Path: "/foo/2", Type: "instance"}},
	}
	for _, test := range tests {
		vol, err := jobs.ParseVolume(test.Input)
		if test.ErrorExpected {
			if err == nil {
				t.Errorf("Expected error in '%s', got none", test.Input)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error in '%s': %#v", test.Input, err)
			} else {
				if !reflect.DeepEqual(test.Expected, vol) {
					t.Errorf("Unexpected result. Expected %#v, got %#v", test.Expected, vol)
				}
			}
		}
	}
}
