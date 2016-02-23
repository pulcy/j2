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

package cluster

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/juju/errgo"

	"github.com/pulcy/j2/util"
)

// ParseClusterFromFile reads a cluster from file
func ParseClusterFromFile(path string) (*Cluster, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, maskAny(err)
	}
	// Parse the input
	root, err := hcl.Parse(string(data))
	if err != nil {
		return nil, maskAny(err)
	}
	// Top-level item should be a list
	list, ok := root.Node.(*ast.ObjectList)
	if !ok {
		return nil, errgo.New("error parsing: root should be an object")
	}
	matches := list.Filter("cluster")
	if len(matches.Items) == 0 {
		return nil, errgo.New("'cluster' stanza not found")
	}

	// Parse hcl into Cluster
	cluster := &Cluster{}
	if err := cluster.parse(matches); err != nil {
		return nil, maskAny(err)
	}
	cluster.setDefaults()

	// Validate the cluster
	if err := cluster.validate(); err != nil {
		return nil, maskAny(err)
	}

	return cluster, nil
}

// Parse a Cluster
func (c *Cluster) parse(list *ast.ObjectList) error {
	list = list.Children()
	if len(list.Items) != 1 {
		return fmt.Errorf("only one 'cluster' block allowed")
	}

	// Get our cluster object
	obj := list.Items[0]

	// Decode the object
	excludeList := []string{
		"default-options",
		"docker",
		"fleet",
	}
	if err := util.Decode(obj.Val, excludeList, nil, c); err != nil {
		return maskAny(err)
	}
	c.Stack = obj.Keys[0].Token.Value().(string)

	// Value should be an object
	var listVal *ast.ObjectList
	if ot, ok := obj.Val.(*ast.ObjectType); ok {
		listVal = ot.List
	} else {
		return errgo.Newf("cluster '%s' value: should be an object", c.Stack)
	}

	// If we have docker object, parse it
	if o := listVal.Filter("docker"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				if err := c.DockerOptions.parse(obj, *c); err != nil {
					return maskAny(err)
				}
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "docker of cluster '%s' is not an object", c.Stack))
			}
		}
	}

	// If we have fleet object, parse it
	if o := listVal.Filter("fleet"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				if err := c.FleetOptions.parse(obj, *c); err != nil {
					return maskAny(err)
				}
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "fleet of cluster '%s' is not an object", c.Stack))
			}
		}
	}

	// Parse default-options
	if o := listVal.Filter("default-options"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			var m map[string]string
			if err := hcl.DecodeObject(&m, o.Val); err != nil {
				return maskAny(err)
			}
			// Merge key/value pairs into myself
			for k, v := range m {
				c.DefaultOptions.SetKV(k, v)
			}
		}
	}

	return nil
}

// parse a DockerOptions
func (options *DockerOptions) parse(obj *ast.ObjectType, c Cluster) error {
	// Parse the object
	excludeList := []string{
		"log-args",
	}
	if err := util.Decode(obj, excludeList, nil, options); err != nil {
		return maskAny(err)
	}
	// Parse log-args
	if o := obj.List.Filter("log-args"); len(o.Items) > 0 {
		list, err := util.ParseStringList(o, fmt.Sprintf("log-args of cluster '%s'", c.Stack))
		if err != nil {
			return maskAny(err)
		}
		options.LoggingArgs = append(options.LoggingArgs, list...)
	}

	return nil
}

// parse a FleetOptions
func (options *FleetOptions) parse(obj *ast.ObjectType, c Cluster) error {
	// Parse the object
	excludeList := []string{
		"after",
		"wants",
		"requires",
	}
	if err := util.Decode(obj, excludeList, nil, options); err != nil {
		return maskAny(err)
	}
	// Parse after
	if o := obj.List.Filter("after"); len(o.Items) > 0 {
		list, err := util.ParseStringList(o, fmt.Sprintf("after of cluster '%s'", c.Stack))
		if err != nil {
			return maskAny(err)
		}
		options.After = list
	}
	// Parse wants
	if o := obj.List.Filter("wants"); len(o.Items) > 0 {
		list, err := util.ParseStringList(o, fmt.Sprintf("wants of cluster '%s'", c.Stack))
		if err != nil {
			return maskAny(err)
		}
		options.Wants = list
	}
	// Parse requires
	if o := obj.List.Filter("requires"); len(o.Items) > 0 {
		list, err := util.ParseStringList(o, fmt.Sprintf("requires of cluster '%s'", c.Stack))
		if err != nil {
			return maskAny(err)
		}
		options.Requires = list
	}

	return nil
}
