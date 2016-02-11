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

package jobs

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/juju/errgo"
)

const (
	protocolTCP = "tcp"
	protocolUDP = "udp"
)

type Port struct {
	Port     string
	Protocol string
}

func (p Port) String() string {
	return fmt.Sprintf("%s/%s", p.Port, p.Protocol)
}

func (d *Port) UnmarshalJSON(data []byte) error {
	if data[0] != '"' {
		newData := []byte{}
		newData = append(newData, '"')
		newData = append(newData, data...)
		newData = append(newData, '"')

		data = newData
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return maskAny(err)
	}

	if err := parseDockerPort(s, d); err != nil {
		return maskAny(err)
	}

	return nil
}

func parseDockerPort(input string, dp *Port) error {
	s := strings.Split(input, "/")

	switch len(s) {
	case 1:
		dp.Port = s[0]
		dp.Protocol = protocolTCP
	case 2:
		dp.Port = s[0]
		dp.Protocol = s[1]
	default:
		return errgo.Newf("Invalid format, must be either <port> or <port>/<prot>, got '%s'", input)
	}

	if parsedPort, err := strconv.Atoi(dp.Port); err != nil {
		return errgo.Notef(err, "Port must be a number, got '%s'", dp.Port)
	} else if parsedPort < 1 || parsedPort > 65535 {
		return errgo.Notef(err, "Port must be a number between 1 and 65535, got '%s'", dp.Port)
	}

	switch dp.Protocol {
	case "":
		return errgo.Newf("Protocol must not be empty.")
	case protocolUDP, protocolTCP:
		return nil
	default:
		return errgo.Newf("Unknown protocol: '%s' in '%s'", dp.Protocol, input)
	}
}
