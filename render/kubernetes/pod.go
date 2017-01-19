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
		p := pod{
			name:  resourceName(tg.Name.String(), ""),
			tasks: tg.Tasks,
		}
		if err := p.validate(); err != nil {
			return nil, maskAny(err)
		}
		return []pod{p}, nil
	}

	taskName2pod := make(map[jobs.TaskName]*pod)
	// First create a pod for all tasks
	for i, t := range tg.Tasks {
		p := &pod{
			index: i,
			tasks: jobs.TaskList{t},
		}
		taskName2pod[t.Name] = p
	}

	group := func(n1, n2 jobs.TaskName) error {
		p1, ok := taskName2pod[n1]
		if !ok {
			return maskAny(fmt.Errorf("Task '%s' not found", n1))
		}
		p2, ok := taskName2pod[n2]
		if !ok {
			return maskAny(fmt.Errorf("Task '%s' not found", n2))
		}
		if p1 == p2 {
			// Already in same pod
			return nil
		}
		p1.tasks = append(p1.tasks, p2.tasks...)
		for _, t := range p2.tasks {
			taskName2pod[t.Name] = p1
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
	for _, p := range taskName2pod {
		if p.name != "" {
			continue
		}
		p.name = createPodName(p, tg)
	}

	// Now build a list of pods
	seenIndexes := make(map[int]struct{})
	var result []pod
	for _, p := range taskName2pod {
		if _, ok := seenIndexes[p.index]; ok {
			continue
		}
		p.sortTasks()
		result = append(result, *p)
		seenIndexes[p.index] = struct{}{}
	}
	sort.Sort(podByIndex(result))

	// Validate pods
	for _, p := range result {
		if err := p.validate(); err != nil {
			return nil, maskAny(err)
		}
	}

	return result, nil
}

// createPodName creates a name for a pod.
// The name consists of the name of the taskgroup followed by the name of the first task (index in taskgroup)
// that is part of this pod.
func createPodName(p *pod, tg *jobs.TaskGroup) string {
	lowestOriginalIndex := -1
	taskName := ""
	for _, t := range p.tasks {
		index := t.OriginalIndex
		if index >= 0 {
			if lowestOriginalIndex < 0 || index < lowestOriginalIndex {
				lowestOriginalIndex = index
				taskName = t.Name.String()
			}
		}
	}
	if lowestOriginalIndex < 0 {
		return resourceName(tg.Name.String(), "")
	}
	return resourceName(tg.Name.String(), "-"+taskName)
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

func (p *pod) validate() error {
	// Pod must have name
	if p.name == "" {
		return maskAny(fmt.Errorf("Pod with tasks %v has no name", p.tasks))
	}
	// All tasks must use same network
	for i := 1; i < len(p.tasks); i++ {
		prev := p.tasks[i-i]
		cur := p.tasks[i]
		if prev.Network != cur.Network {
			return maskAny(fmt.Errorf("Cannot mix different networks in a single pod. (tasks %s and %s)", prev.FullName(), cur.FullName()))
		}
	}
	// Cannot have oneshot & service tasks in 1 pod
	if p.hasOneShotTasks() && p.hasServiceTasks() {
		return maskAny(fmt.Errorf("Cannot mix oneshot & service tasks in a single pod."))
	}
	return nil
}

// hasServiceTasks returns true if there is at least 1 task of type Service in this pod.
func (p *pod) hasServiceTasks() bool {
	for _, t := range p.tasks {
		if t.Type.IsService() {
			return true
		}
	}
	return false
}

// hasOneShotTasks returns true if there is at least 1 task of type OneShot in this pod.
func (p *pod) hasOneShotTasks() bool {
	for _, t := range p.tasks {
		if t.Type.IsOneshot() {
			return true
		}
	}
	return false
}

// hasRWHostVolumes returns true if there is at least 1 task that has a volume mapped to a host folder and is read/write.
func (p *pod) hasRWHostVolumes() bool {
	for _, t := range p.tasks {
		for _, v := range t.Volumes {
			if v.IsLocal() && !v.IsReadOnly() {
				return true
			}
		}
	}
	return false
}
