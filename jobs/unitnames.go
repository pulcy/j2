package jobs

import (
	"fmt"
	"strings"
)

func IsUnitForJob(unitName string, jobName JobName) bool {
	prefix := fmt.Sprintf("%s-", jobName)
	return strings.HasPrefix(unitName, prefix) && strings.HasSuffix(unitName, ".service")
}

func IsUnitForTaskGroup(unitName string, jobName JobName, taskGroupName TaskGroupName) bool {
	prefix := fmt.Sprintf("%s-%s-", jobName, taskGroupName)
	return strings.HasPrefix(unitName, prefix) && strings.HasSuffix(unitName, ".service")
}

func IsUnitForTask(unitName string, jobName JobName, taskGroupName TaskGroupName, taskName TaskName) bool {
	prefix := fmt.Sprintf("%s-%s-%s", jobName, taskGroupName, taskName)
	if strings.HasPrefix(unitName, prefix) && strings.HasSuffix(unitName, ".service") {
		remainder := unitName[len(prefix):]
		return strings.HasPrefix(remainder, "@") || strings.HasPrefix(remainder, ".")
	} else {
		return false
	}
}
