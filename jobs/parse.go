package jobs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	hclobj "github.com/hashicorp/hcl/hcl"
	"github.com/mitchellh/mapstructure"
)

// ParseJob takes input from a given reader and parses it into a Job.
func ParseJob(input []byte) (*Job, error) {

	// Parse the input
	obj, err := hcl.Parse(string(input))
	if err != nil {
		return nil, maskAny(err)
	}

	// Parse hcl into Job
	job := &Job{}
	if err := job.parse(obj); err != nil {
		return nil, maskAny(err)
	}

	json, _ := json.MarshalIndent(job, "", "  ")
	fmt.Printf("job:\n%s\n", json)

	fmt.Printf("jobObj:\n%#v\n", obj)

	// Validate the job
	if err := job.Validate(); err != nil {
		return nil, maskAny(err)
	}

	// Link internal structures
	job.link()

	return job, nil
}

// ParseJobFromFile reads a job from file
func ParseJobFromFile(path string) (*Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, maskAny(err)
	}
	job, err := ParseJob(data)
	if err != nil {
		return nil, maskAny(err)
	}
	return job, nil
}

func (j *Job) parse(obj *hclobj.Object) error {
	// Decode the full thing into a map[string]interface for ease
	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, obj); err != nil {
		return maskAny(err)
	}

	// Decode the rest
	if err := mapstructure.WeakDecode(m, j); err != nil {
		return maskAny(err)
	}

	// If we have tasks outside, do those
	if o := obj.Get("tasks", false); o != nil {
		tmp := &TaskGroup{}
		if err := tmp.parseTasks(o); err != nil {
			return err
		}

		for _, t := range tmp.Tasks {
			tg := &TaskGroup{
				Name:  TaskGroupName(t.Name),
				Count: 1,
				Tasks: []*Task{t},
			}
			j.Groups = append(j.Groups, tg)
		}
	}

	// Parse the task groups
	if o := obj.Get("groups", false); o != nil {
		if err := j.parseGroups(o); err != nil {
			return fmt.Errorf("error parsing 'group': %s", err)
		}
	}

	return nil
}

func (j *Job) parseGroups(obj *hclobj.Object) error {
	// Get all the maps of keys to the actual object
	objects := make(map[string]*hclobj.Object)
	for _, o1 := range obj.Elem(false) {
		for _, o2 := range o1.Elem(true) {
			if _, ok := objects[o2.Key]; ok {
				return fmt.Errorf(
					"group '%s' defined more than once",
					o2.Key)
			}

			objects[o2.Key] = o2
		}
	}

	if len(objects) == 0 {
		return nil
	}

	// Go through each object and turn it into an actual result.
	collection := []*TaskGroup{}
	for n, o := range objects {
		var m map[string]interface{}
		if err := hcl.DecodeObject(&m, o); err != nil {
			return maskAny(err)
		}

		// Default count to 1 if not specified
		if _, ok := m["count"]; !ok {
			m["count"] = defaultCount
		}

		// Build the group with the basic decode
		tg := &TaskGroup{}
		tg.Name = TaskGroupName(n)
		if err := mapstructure.WeakDecode(m, tg); err != nil {
			return maskAny(err)
		}

		// Parse tasks
		if o := o.Get("tasks", false); o != nil {
			if err := tg.parseTasks(o); err != nil {
				return maskAny(err)
			}
		}

		collection = append(collection, tg)
	}

	j.Groups = append(j.Groups, collection...)
	return nil
}

func (tg *TaskGroup) parseTasks(obj *hclobj.Object) error {
	// Get all the maps of keys to the actual object
	objects := make([]*hclobj.Object, 0, 5)
	set := make(map[string]struct{})
	for _, o1 := range obj.Elem(false) {
		for _, o2 := range o1.Elem(true) {
			if _, ok := set[o2.Key]; ok {
				return fmt.Errorf(
					"group '%s' defined more than once",
					o2.Key)
			}

			objects = append(objects, o2)
			set[o2.Key] = struct{}{}
		}
	}

	if len(objects) == 0 {
		return nil
	}

	for _, o := range objects {
		var m map[string]interface{}
		if err := hcl.DecodeObject(&m, o); err != nil {
			return err
		}
		delete(m, "env")

		// Build the task
		t := &Task{}
		t.Name = TaskName(o.Key)
		if err := mapstructure.WeakDecode(m, t); err != nil {
			return maskAny(err)
		}

		// If we have env, then parse them
		if o := o.Get("env", false); o != nil {
			for _, o := range o.Elem(false) {
				var m map[string]interface{}
				if err := hcl.DecodeObject(&m, o); err != nil {
					return maskAny(err)
				}
				if err := mapstructure.WeakDecode(m, &t.Environment); err != nil {
					return maskAny(err)
				}
			}
		}

		tg.Tasks = append(tg.Tasks, t)
	}

	return nil
}
