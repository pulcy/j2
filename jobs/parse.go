package jobs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/hashicorp/hcl"
	hclobj "github.com/hashicorp/hcl/hcl"
	"github.com/juju/errgo"
	"github.com/mitchellh/mapstructure"

	fg "arvika.pulcy.com/pulcy/deployit/flags"
)

// ParseJob takes input from a given reader and parses it into a Job.
func parseJob(input []byte, jf *jobFunctions) (*Job, error) {
	// Create a template, add the function map, and parse the text.
	tmpl, err := template.New("job").Funcs(jf.Functions()).Parse(string(input))
	if err != nil {
		return nil, maskAny(err)
	}

	// Run the template to verify the output.
	buffer := &bytes.Buffer{}
	err = tmpl.Execute(buffer, nil)
	if err != nil {
		return nil, maskAny(err)
	}

	// Parse the input
	obj, err := hcl.Parse(buffer.String())
	if err != nil {
		return nil, maskAny(err)
	}

	// Parse hcl into Job
	job := &Job{}
	if err := job.parse(obj); err != nil {
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
func ParseJobFromFile(path string, options fg.Options) (*Job, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, maskAny(err)
	}
	jf := newJobFunctions(path, options)
	job, err := parseJob(data, jf)
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
	delete(m, "group")
	delete(m, "task")

	// Decode the rest
	if err := mapstructure.WeakDecode(m, j); err != nil {
		return maskAny(err)
	}

	// If we have tasks outside, do those
	if o := obj.Get("task", false); o != nil {
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
	if o := obj.Get("group", false); o != nil {
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
	for _, o := range objects {
		// Build the group with the basic decode
		tg := &TaskGroup{}
		tg.Name = TaskGroupName(o.Key)
		if err := tg.parse(o); err != nil {
			return maskAny(err)
		}

		j.Groups = append(j.Groups, tg)
	}

	return nil
}

// parse a task group
func (tg *TaskGroup) parse(obj *hclobj.Object) error {
	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, obj); err != nil {
		return maskAny(err)
	}
	delete(m, "task")

	// Default count to 1 if not specified
	if _, ok := m["count"]; !ok {
		m["count"] = defaultCount
	}

	// Build the group with the basic decode
	if err := mapstructure.WeakDecode(m, tg); err != nil {
		return maskAny(err)
	}

	// Parse tasks
	if o := obj.Get("task", false); o != nil {
		if err := tg.parseTasks(o); err != nil {
			return maskAny(err)
		}
	}

	return nil
}

// parse a list of tasks
func (tg *TaskGroup) parseTasks(obj *hclobj.Object) error {
	// Get all the maps of keys to the actual object
	objects := make([]*hclobj.Object, 0, 5)
	set := make(map[string]struct{})
	for _, o1 := range obj.Elem(false) {
		for _, o2 := range o1.Elem(true) {
			if _, ok := set[o2.Key]; ok {
				return fmt.Errorf(
					"task '%s' defined more than once",
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
		t := &Task{}
		t.Name = TaskName(o.Key)
		if err := t.parse(o); err != nil {
			return maskAny(err)
		}

		tg.Tasks = append(tg.Tasks, t)
	}

	return nil
}

// parse a task
func (t *Task) parse(obj *hclobj.Object) error {
	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, obj); err != nil {
		return err
	}
	delete(m, "env")
	delete(m, "image")
	delete(m, "volumes")
	delete(m, "volumes-from")
	delete(m, "frontend")

	// Default count to 1 if not specified
	if _, ok := m["count"]; !ok {
		m["count"] = defaultCount
	}

	// Build the task
	if err := mapstructure.WeakDecode(m, t); err != nil {
		return maskAny(err)
	}

	if o := obj.Get("image", false); o != nil && o.Type == hclobj.ValueTypeString {
		img, err := ParseDockerImage(o.Value.(string))
		if err != nil {
			return maskAny(err)
		}
		t.Image = img
	} else if o != nil {
		return maskAny(errgo.WithCausef(nil, ValidationError, "image of task %s is not a string", t.Name))
	} else {
		return maskAny(errgo.WithCausef(nil, ValidationError, "image missing for task %s", t.Name))
	}

	// If we have env, then parse them
	if o := obj.Get("env", false); o != nil {
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

	// Parse volumes
	if o := obj.Get("volumes", false); o != nil {
		if o.Type == hclobj.ValueTypeString {
			t.Volumes = []string{o.Value.(string)}
		} else if o.Type == hclobj.ValueTypeList {
			for _, o := range o.Elem(false) {
				if o.Type == hclobj.ValueTypeString {
					t.Volumes = append(t.Volumes, o.Value.(string))
				} else {
					return maskAny(errgo.WithCausef(nil, ValidationError, "element of volumes array of task %s is not a string", t.Name))
				}
			}
		} else {
			return maskAny(errgo.WithCausef(nil, ValidationError, "volumes of task %s is not a string or array", t.Name))
		}
	}

	// Parse volumes-from
	if o := obj.Get("volumes-from", false); o != nil {
		if o.Type == hclobj.ValueTypeString {
			t.VolumesFrom = []TaskName{TaskName(o.Value.(string))}
		} else if o.Type == hclobj.ValueTypeList {
			for _, o := range o.Elem(false) {
				if o.Type == hclobj.ValueTypeString {
					t.VolumesFrom = append(t.VolumesFrom, TaskName(o.Value.(string)))
				} else {
					return maskAny(errgo.WithCausef(nil, ValidationError, "element of volumes-from array of task %s is not a string", t.Name))
				}
			}
		} else {
			return maskAny(errgo.WithCausef(nil, ValidationError, "volumes-from of task %s is not a string or array", t.Name))
		}
	}

	// Parse frontends

	if first := obj.Get("frontend", false); first != nil {
		for _, o := range first.Elem(false) {
			if o.Type == hclobj.ValueTypeObject {
				f := FrontEnd{}
				if err := f.parse(o); err != nil {
					return maskAny(err)
				}
				t.FrontEnds = append(t.FrontEnds, f)
			} else {
				return maskAny(errgo.WithCausef(nil, ValidationError, "frontend of task %s is not an object or array", t.Name))
			}
		}
	}

	return nil
}

// parse a frontend
func (f *FrontEnd) parse(obj *hclobj.Object) error {
	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, obj); err != nil {
		return err
	}

	// Build the frontend
	if err := mapstructure.WeakDecode(m, f); err != nil {
		return maskAny(err)
	}

	return nil
}
