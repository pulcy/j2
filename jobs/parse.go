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
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/hashicorp/hcl/hcl/token"
	"github.com/juju/errgo"
	"github.com/mitchellh/mapstructure"

	fg "github.com/pulcy/deployit/flags"
)

type parseJobOptions struct {
	Cluster fg.Cluster
}

// ParseJob takes input from a given reader and parses it into a Job.
func parseJob(input []byte, opts parseJobOptions, jf *jobFunctions) (*Job, error) {
	// Create a template, add the function map, and parse the text.
	tmpl, err := template.New("job").Funcs(jf.Functions()).Parse(string(input))
	if err != nil {
		return nil, maskAny(err)
	}

	// Run the template to verify the output.
	buffer := &bytes.Buffer{}
	err = tmpl.Execute(buffer, opts)
	if err != nil {
		return nil, maskAny(err)
	}

	// Parse the input
	root, err := hcl.Parse(buffer.String())
	if err != nil {
		return nil, maskAny(err)
	}
	// Top-level item should be a list
	list, ok := root.Node.(*ast.ObjectList)
	if !ok {
		return nil, errgo.New("error parsing: root should be an object")
	}

	// Parse hcl into Job
	job := &Job{}
	matches := list.Filter("job")
	if len(matches.Items) == 0 {
		return nil, errgo.New("'job' stanza not found")
	}
	if err := job.parse(matches); err != nil {
		return nil, maskAny(err)
	}

	// Link internal structures
	job.link()

	// Validate the job
	if err := job.Validate(); err != nil {
		return nil, maskAny(err)
	}

	return job, nil
}

// ParseJobFromFile reads a job from file
func ParseJobFromFile(path string, cluster fg.Cluster, options fg.Options) (*Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, maskAny(err)
	}
	jf := newJobFunctions(path, cluster, options)
	opts := parseJobOptions{
		Cluster: cluster,
	}
	job, err := parseJob(data, opts, jf)
	if err != nil {
		return nil, maskAny(err)
	}
	return job, nil
}

func (j *Job) parse(list *ast.ObjectList) error {
	list = list.Children()
	if len(list.Items) != 1 {
		return fmt.Errorf("only one 'job' block allowed")
	}

	// Get our job object
	obj := list.Items[0]

	// Decode the object
	if err := decode(obj.Val, []string{"group", "task", "constraint"}, nil, j); err != nil {
		return maskAny(err)
	}

	j.Name = JobName(obj.Keys[0].Token.Value().(string))

	// Value should be an object
	var listVal *ast.ObjectList
	if ot, ok := obj.Val.(*ast.ObjectType); ok {
		listVal = ot.List
	} else {
		return errgo.Newf("job '%s' value: should be an object", j.Name)
	}

	// If we have tasks outside, do those
	if o := listVal.Filter("task"); len(o.Items) > 0 {
		tmp := &TaskGroup{}
		if err := tmp.parseTasks(o); err != nil {
			return err
		}

		for _, t := range tmp.Tasks {
			tg := &TaskGroup{
				Name:   TaskGroupName(t.Name),
				Count:  t.Count,
				Global: t.Global,
				Tasks:  []*Task{t},
			}
			j.Groups = append(j.Groups, tg)
		}
	}

	// Parse the task groups
	if o := listVal.Filter("group"); len(o.Items) > 0 {
		if err := j.parseGroups(o); err != nil {
			return fmt.Errorf("error parsing 'group': %s", err)
		}
	}

	// Parse constraints
	if o := listVal.Filter("constraint"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				c := Constraint{}
				if err := c.parse(obj); err != nil {
					return maskAny(err)
				}
				j.Constraints = append(j.Constraints, c)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "constraint of job %s is not an object", j.Name))
			}
		}
	}

	return nil
}

func (j *Job) parseGroups(list *ast.ObjectList) error {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil
	}

	seen := make(map[string]struct{})
	for _, item := range list.Items {
		n := item.Keys[0].Token.Value().(string)

		// Make sure we haven't already found this
		if _, ok := seen[n]; ok {
			return fmt.Errorf("group '%s' defined more than once", n)
		}
		seen[n] = struct{}{}

		// We need this later
		obj, ok := item.Val.(*ast.ObjectType)
		if !ok {
			return fmt.Errorf("group '%s': should be an object", n)
		}

		// Build the group with the basic decode
		tg := &TaskGroup{}
		tg.Name = TaskGroupName(n)
		if err := tg.parse(obj); err != nil {
			return maskAny(err)
		}

		j.Groups = append(j.Groups, tg)
	}

	return nil
}

// parse a task group
func (tg *TaskGroup) parse(obj *ast.ObjectType) error {
	// Build the group with the basic decode
	defaultValues := map[string]interface{}{
		"count": defaultCount,
	}
	if err := decode(obj, []string{"task", "constraint"}, defaultValues, tg); err != nil {
		return maskAny(err)
	}

	// Parse tasks
	if o := obj.List.Filter("task"); len(o.Items) > 0 {
		if err := tg.parseTasks(o); err != nil {
			return maskAny(err)
		}
	}

	// Parse constraints
	if o := obj.List.Filter("constraint"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				c := Constraint{}
				if err := c.parse(obj); err != nil {
					return maskAny(err)
				}
				tg.Constraints = append(tg.Constraints, c)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "constraint of task-group %s is not an object", tg.Name))
			}
		}
	}

	return nil
}

// parse a list of tasks
func (tg *TaskGroup) parseTasks(list *ast.ObjectList) error {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil
	}

	// Get all the maps of keys to the actual object
	seen := make(map[string]struct{})
	for _, item := range list.Items {
		n := item.Keys[0].Token.Value().(string)
		if _, ok := seen[n]; ok {
			return fmt.Errorf("task '%s' defined more than once", n)
		}
		seen[n] = struct{}{}
		obj, ok := item.Val.(*ast.ObjectType)
		if !ok {
			return fmt.Errorf("task '%s': should be an object", tg.Name)
		}

		t := &Task{}
		t.Name = TaskName(n)
		if err := t.parse(obj); err != nil {
			return maskAny(err)
		}

		tg.Tasks = append(tg.Tasks, t)
	}

	return nil
}

// parse a task
func (t *Task) parse(obj *ast.ObjectType) error {
	// Build the task
	excludedKeys := []string{
		"env",
		"image",
		"volumes",
		"volumes-from",
		"frontend",
		"private-frontend",
		"capabilities",
		"links",
		"secret",
		"constraint",
	}
	defaultValues := map[string]interface{}{
		"count": defaultCount,
	}
	if err := decode(obj, excludedKeys, defaultValues, t); err != nil {
		return maskAny(err)
	}

	if o := obj.List.Filter("image"); len(o.Items) > 0 {
		if len(o.Items) > 1 {
			return maskAny(errgo.WithCausef(nil, ValidationError, "task %s defines multiple images", t.Name))
		}
		if obj, ok := o.Items[0].Val.(*ast.LiteralType); ok && obj.Token.Type == token.STRING {
			img, err := ParseDockerImage(obj.Token.Value().(string))
			if err != nil {
				return maskAny(err)
			}
			t.Image = img
		} else {
			return maskAny(errgo.WithCausef(nil, ValidationError, "image for task %s is not a string", t.Name))
		}
	} else {
		return maskAny(errgo.WithCausef(nil, ValidationError, "image missing for task %s", t.Name))
	}

	// If we have env, then parse them
	if o := obj.List.Filter("env"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			if err := decode(o.Val, nil, nil, &t.Environment); err != nil {
				return maskAny(err)
			}
		}
	}

	// Parse volumes
	if o := obj.List.Filter("volumes"); len(o.Items) > 0 {
		list, err := parseStringList(o, fmt.Sprintf("volumes of task %s", t.Name))
		if err != nil {
			return maskAny(err)
		}
		t.Volumes = list
	}

	// Parse volumes-from
	if o := obj.List.Filter("volumes-from"); len(o.Items) > 0 {
		list, err := parseStringList(o, fmt.Sprintf("volumes-from of task %s", t.Name))
		if err != nil {
			return maskAny(err)
		}
		for _, x := range list {
			t.VolumesFrom = append(t.VolumesFrom, TaskName(x))
		}
	}

	// Parse capabilities
	if o := obj.List.Filter("capabilities"); len(o.Items) > 0 {
		list, err := parseStringList(o, fmt.Sprintf("capabilities of task %s", t.Name))
		if err != nil {
			return maskAny(err)
		}
		t.Capabilities = list
	}

	// Parse links
	if o := obj.List.Filter("links"); len(o.Items) > 0 {
		list, err := parseStringList(o, fmt.Sprintf("links of task %s", t.Name))
		if err != nil {
			return maskAny(err)
		}
		for _, x := range list {
			t.Links = append(t.Links, LinkName(x).normalize())
		}
	}

	// Parse public frontends
	if o := obj.List.Filter("frontend"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				f := PublicFrontEnd{}
				if err := f.parse(obj); err != nil {
					return maskAny(err)
				}
				t.PublicFrontEnds = append(t.PublicFrontEnds, f)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "frontend of task %s is not an object or array", t.Name))
			}
		}
	}

	// Parse private frontends
	if o := obj.List.Filter("private-frontend"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				f := PrivateFrontEnd{}
				if err := f.parse(obj); err != nil {
					return maskAny(err)
				}
				t.PrivateFrontEnds = append(t.PrivateFrontEnds, f)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "private-frontend of task %s is not an object or array", t.Name))
			}
		}
	}

	// Parse secrets
	if o := obj.List.Filter("secret"); len(o.Items) > 0 {
		for _, o := range o.Children().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				s := Secret{}
				n := o.Keys[0].Token.Value().(string)
				if err := s.parse(obj); err != nil {
					return maskAny(err)
				}
				s.Path = n
				t.Secrets = append(t.Secrets, s)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "secret of task %s is not an object or array", t.Name))
			}
		}
	}

	// Parse constraints
	if o := obj.List.Filter("constraint"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				c := Constraint{}
				if err := c.parse(obj); err != nil {
					return maskAny(err)
				}
				t.Constraints = append(t.Constraints, c)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "constraint of task %s is not an object", t.Name))
			}
		}
	}

	return nil
}

// parse a public frontend
func (f *PublicFrontEnd) parse(obj *ast.ObjectType) error {
	// Build the frontend
	excludedKeys := []string{
		"user",
	}
	if err := decode(obj, excludedKeys, nil, f); err != nil {
		return maskAny(err)
	}
	if o := obj.List.Filter("user"); len(o.Items) > 0 {
		for _, o := range o.Children().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				n := o.Keys[0].Token.Value().(string)
				u := User{Name: n}
				if err := u.parse(obj); err != nil {
					return maskAny(err)
				}
				f.Users = append(f.Users, u)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "user of frontend %#v is not an object or array", f))
			}
		}
	}

	return nil
}

// parse a private frontend
func (f *PrivateFrontEnd) parse(obj *ast.ObjectType) error {
	// Build the frontend
	excludedKeys := []string{
		"user",
	}
	defaultValues := map[string]interface{}{
		"port": 80,
	}
	if err := decode(obj, excludedKeys, defaultValues, f); err != nil {
		return maskAny(err)
	}
	if o := obj.List.Filter("user"); len(o.Items) > 0 {
		for _, o := range o.Children().Items {
			if obj, ok := o.Val.(*ast.ObjectType); ok {
				n := o.Keys[0].Token.Value().(string)
				u := User{Name: n}
				if err := u.parse(obj); err != nil {
					return maskAny(err)
				}
				f.Users = append(f.Users, u)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "user of frontend %#v is not an object or array", f))
			}
		}
	}

	return nil
}

// parse a constraint
func (c *Constraint) parse(obj *ast.ObjectType) error {
	// Build the constraint
	if err := decode(obj, nil, nil, c); err != nil {
		return maskAny(err)
	}
	return nil
}

// parse a secret
func (s *Secret) parse(obj *ast.ObjectType) error {
	// Build the secret
	if err := decode(obj, nil, nil, s); err != nil {
		return maskAny(err)
	}

	return nil
}

// parse a user
func (u *User) parse(obj *ast.ObjectType) error {
	// Build the secret
	if err := decode(obj, nil, nil, u); err != nil {
		return maskAny(err)
	}

	return nil
}

func parseStringList(o *ast.ObjectList, context string) ([]string, error) {
	result := []string{}
	for _, o := range o.Elem().Items {
		if olit, ok := o.Val.(*ast.LiteralType); ok && olit.Token.Type == token.STRING {
			result = append(result, olit.Token.Value().(string))
		} else if list, ok := o.Val.(*ast.ListType); ok {
			for _, n := range list.List {
				if olit, ok := n.(*ast.LiteralType); ok && olit.Token.Type == token.STRING {
					result = append(result, olit.Token.Value().(string))
				} else {
					return nil, maskAny(errgo.WithCausef(nil, ValidationError, "element of %s is not a string but %v", context, n))
				}
			}
		} else {
			return nil, maskAny(errgo.WithCausef(nil, ValidationError, "%s is not a string or array", context))
		}
	}
	return result, nil
}

// Decode from object to data structure using `mapstructure`
func decode(obj ast.Node, excludeKeys []string, defaultValues map[string]interface{}, data interface{}) error {
	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, obj); err != nil {
		return maskAny(err)
	}
	for _, key := range excludeKeys {
		delete(m, key)
	}
	for k, v := range defaultValues {
		if _, ok := m[k]; !ok {
			m[k] = v
		}
	}
	decoderConfig := &mapstructure.DecoderConfig{
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Metadata:         nil,
		Result:           data,
	}
	decoder, err := mapstructure.NewDecoder(decoderConfig)
	if err != nil {
		return maskAny(err)
	}
	return maskAny(decoder.Decode(m))
}
