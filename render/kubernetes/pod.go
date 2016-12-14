package kubernetes

import (
	"fmt"
	"sort"

	"github.com/pulcy/j2/jobs"
)

type pod struct {
	index int // Temporary index used for building
	name  string
	tasks jobs.TaskList
}

// groupTaskIntoPods groups all tasks (of a task group) into pods
// such that:
// - VolumesFrom are honored
// - RestartPolicy is honored
func groupTaskIntoPods(tg *jobs.TaskGroup) ([]pod, error) {
	if tg.RestartPolicy.IsAll() {
		// Put everything into 1 pod
		return []pod{
			pod{
				tasks: tg.Tasks,
			},
		}, nil
	}

	name2pod := make(map[jobs.TaskName]*pod)
	// First create a pod for all tasks
	for i, t := range tg.Tasks {
		p := &pod{
			index: i,
			tasks: jobs.TaskList{t},
		}
		name2pod[t.Name] = p
	}

	group := func(n1, n2 jobs.TaskName) error {
		p1, ok := name2pod[n1]
		if !ok {
			return maskAny(fmt.Errorf("Task '%s' not found", n1))
		}
		p2, ok := name2pod[n2]
		if !ok {
			return maskAny(fmt.Errorf("Task '%s' not found", n2))
		}
		if p1 == p2 {
			// Already in same pod
			return nil
		}
		p1.tasks = append(p1.tasks, p2.tasks...)
		for _, t := range p2.tasks {
			name2pod[t.Name] = p1
		}
		return nil
	}

	for _, t := range tg.Tasks {
		// Group by VolumesFrom
		for _, from := range t.VolumesFrom {
			if err := group(t.Name, from); err != nil {
				return nil, maskAny(err)
			}
		}
		// Group by After
		for _, from := range t.After {
			if err := group(t.Name, from); err != nil {
				return nil, maskAny(err)
			}
		}
	}

	// Assign names to the pods
	for _, p := range name2pod {
		if p.name != "" {
			continue
		}
		if len(tg.Tasks) == 1 {
			p.name = resourceName(tg.Name.String(), kindPod)
		} else {
			p.name = resourceName(tg.Name.String(), fmt.Sprintf("%s-%d", kindPod, p.index))
		}
	}

	// Now build a list of pods
	seenIndexes := make(map[int]struct{})
	var result []pod
	for _, p := range name2pod {
		if _, ok := seenIndexes[p.index]; ok {
			continue
		}
		p.sortTasks()
		result = append(result, *p)
		seenIndexes[p.index] = struct{}{}
	}
	sort.Sort(podByIndex(result))

	return result, nil
}

type podByIndex []pod

func (l podByIndex) Len() int           { return len(l) }
func (l podByIndex) Less(i, j int) bool { return l[i].index < l[j].index }
func (l podByIndex) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }

// sortTasks orders the tasks in the pod such that After relations are honored.
func (p *pod) sortTasks() {
	// Sort by name first
	sort.Sort(p.tasks)

	for i := 0; i < len(p.tasks); {
		t := p.tasks[i]
		moved := false
		for _, name := range t.After {
			otherIndex := p.tasks.IndexByName(name)
			if otherIndex > i {
				p.tasks.Swap(i, otherIndex)
				moved = true
				break
			}
		}
		if !moved {
			i++
		}
	}
}
