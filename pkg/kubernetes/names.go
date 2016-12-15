package kubernetes

import "strings"

var (
	resourceNameReplacer = strings.NewReplacer(
		"/", "-",
		"_", "-",
	)
)

// ResourceName replaces all characters in the given name that are not valid for K8S resource names.
func ResourceName(fullName string) string {
	return resourceNameReplacer.Replace(fullName)
}
