// Copyright 2014 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestValidateContentType(t *testing.T) {
	tests := []struct {
		ctypes []string
		valid  bool
	}{
		{[]string{"application/json"}, true},
		{[]string{"application/json; quality=0.9"}, true},

		{[]string{"application/xml"}, false},
		{[]string{"application/xml; quality=0.8"}, false},
		{[]string{"application/json", "application/json"}, false},
		{[]string{"application/json", "application/xml"}, false},
	}

	for i, tt := range tests {
		req := http.Request{Header: http.Header{"Content-Type": tt.ctypes}}
		err := validateContentType(&req)
		if (err == nil) != tt.valid {
			t.Errorf("case %d: expected valid=%t, got %v", i, tt.valid, err)
		}
	}
}

func TestSendResponseMarshalSuccess(t *testing.T) {
	rw := httptest.NewRecorder()
	sendResponse(rw, http.StatusOK, nil)

	if rw.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", rw.Code)
	}

	body := rw.Body.String()
	expect := "null"
	if body != expect {
		t.Errorf("Expected body %q, got %q", expect, body)
	}
}

func TestSendResponseMarshalFailure(t *testing.T) {
	rw := httptest.NewRecorder()

	// channels are not JSON-serializable
	sendResponse(rw, http.StatusOK, make(chan bool))

	if rw.Code != http.StatusInternalServerError {
		t.Errorf("Expected 500, got %d", rw.Code)
	}

	if rw.Body.Len() != 0 {
		t.Errorf("Expected empty response body")
	}
}

func TestSendError(t *testing.T) {
	rw := httptest.NewRecorder()
	sendError(rw, http.StatusBadRequest, errors.New("sentinel"))

	if rw.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", rw.Code)
	}

	body := rw.Body.String()
	expect := `{"error":{"code":400,"message":"sentinel"}}`
	if body != expect {
		t.Errorf("Expected body %q, got %q", expect, body)
	}

	ctypes := rw.HeaderMap["Content-Type"]
	expectCTypes := []string{"application/json"}
	if !reflect.DeepEqual(ctypes, expectCTypes) {
		t.Errorf("Expected header Content-Type to be %v, got %v", expectCTypes, ctypes)
	}
}

func TestSendNilError(t *testing.T) {
	rw := httptest.NewRecorder()
	sendError(rw, http.StatusBadRequest, nil)

	body := rw.Body.String()
	expect := `{"error":{"code":400,"message":""}}`
	if body != expect {
		t.Errorf("Expected body %q, got %q", expect, body)
	}
}
