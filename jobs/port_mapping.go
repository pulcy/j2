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
	"fmt"
	"strconv"
	"strings"

	"github.com/juju/errgo"
)

type PortMapping string

// Parse parses a given port mapping.
func (p PortMapping) Parse() (ParsedPortMapping, error) {
	result, err := parsePortMapping(string(p))
	if err != nil {
		return ParsedPortMapping{}, maskAny(err)
	}
	if err := result.Validate(); err != nil {
		return ParsedPortMapping{}, maskAny(err)
	}
	return result, nil
}

func (p PortMapping) String() string {
	return string(p)
}

type ParsedPortMapping struct {
	HostIP        string
	HostPort      int
	ContainerPort int
	Protocol      string
}

func (p ParsedPortMapping) HasHostIP() bool {
	return p.HostIP != ""
}

func (p ParsedPortMapping) HasHostPort() bool {
	return p.HostPort != 0
}

func (p ParsedPortMapping) IsTCP() bool {
	return p.Protocol == "" || p.Protocol == protocolTCP
}

func (p ParsedPortMapping) IsUDP() bool {
	return p.Protocol == protocolUDP
}

func (p ParsedPortMapping) ProtocolString() string {
	if p.IsTCP() {
		return "TCP"
	}
	if p.IsUDP() {
		return "UDP"
	}
	return p.Protocol
}

func (p ParsedPortMapping) String() string {
	containerPort := strconv.Itoa(p.ContainerPort)
	if p.Protocol == protocolUDP {
		containerPort = containerPort + "/" + p.Protocol
	}
	hostPort := ""
	if p.HostPort != 0 {
		hostPort = strconv.Itoa(p.HostPort)
	}
	if p.HostIP == "" {
		return hostPort + ":" + containerPort
	}
	return p.HostIP + ":" + hostPort + ":" + containerPort
}

// Validate checks the port mapping for errors
func (p ParsedPortMapping) Validate() error {
	if p.HostPort < 0 || p.HostPort > 65535 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "HostPort must be a number between 0 and 65535, got '%s'", p.HostPort))
	}
	if p.ContainerPort < 1 || p.ContainerPort > 65535 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "ContainerPort must be a number between 1 and 65535, got '%s'", p.ContainerPort))
	}

	switch p.Protocol {
	case "":
		return maskAny(errgo.WithCausef(nil, ValidationError, "Protocol must not be empty."))
	case protocolUDP, protocolTCP:
	// OK
	default:
		return maskAny(errgo.WithCausef(nil, ValidationError, "Unknown protocol: '%s'", p.Protocol))
	}
	return nil
}

func parsePortMapping(input string) (ParsedPortMapping, error) {
	parts := strings.Split(input, ":")
	hostIP := ""
	containerPort := 0
	protocol := ""
	hostPort := 0
	var err error
	switch len(parts) {
	case 1:
		containerPort, protocol, err = parseContainerPort(parts[0])
		if err != nil {
			return ParsedPortMapping{}, maskAny(err)
		}
	case 2:
		if parts[0] != "" {
			hostPort, err = strconv.Atoi(parts[0])
			if err != nil {
				return ParsedPortMapping{}, maskAny(err)
			}
		}
		containerPort, protocol, err = parseContainerPort(parts[1])
		if err != nil {
			return ParsedPortMapping{}, maskAny(err)
		}
	case 3:
		if parts[0] != "" {
			hostIP = parts[0]
		}
		if parts[1] != "" {
			hostPort, err = strconv.Atoi(parts[1])
			if err != nil {
				return ParsedPortMapping{}, maskAny(err)
			}
		}
		containerPort, protocol, err = parseContainerPort(parts[2])
		if err != nil {
			return ParsedPortMapping{}, maskAny(err)
		}
	default:
		return ParsedPortMapping{}, maskAny(fmt.Errorf("Unknown port format '%s'", input))
	}
	return ParsedPortMapping{
		HostIP:        hostIP,
		HostPort:      hostPort,
		ContainerPort: containerPort,
		Protocol:      protocol,
	}, nil
}

// parseContainerPort parses input like "port" or "port/protocol".
func parseContainerPort(input string) (int, string, error) {
	s := strings.Split(input, "/")
	var portStr, protocol string
	switch len(s) {
	case 1:
		portStr = s[0]
		protocol = protocolTCP
	case 2:
		portStr = s[0]
		protocol = strings.ToLower(s[1])
	default:
		return 0, "", maskAny(fmt.Errorf("Invalid format, must be either <port> or <port>/<prot>, got '%s'", input))
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return 0, "", maskAny(fmt.Errorf("Port must be a number, got '%s'", portStr))
	} else if port < 1 || port > 65535 {
		return 0, "", maskAny(fmt.Errorf("Port must be a number between 1 and 65535, got '%s'", portStr))
	}

	switch protocol {
	case "":
		return 0, "", maskAny(fmt.Errorf("Protocol must not be empty."))
	case protocolUDP, protocolTCP:
		return port, protocol, nil
	default:
		return 0, "", maskAny(fmt.Errorf("Unknown protocol: '%s' in '%s'", protocol, input))
	}
}
