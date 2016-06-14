package cmdline

import (
	"fmt"
	"strings"
)

type Cmdline struct {
	cmd          []string
	AllowFailure bool
}

const (
	shortArgLen = 4
)

// New creates a new Cmdline filled with the given initial arguments
func New(env map[string]string, args ...string) *Cmdline {
	cmd := &Cmdline{}
	return cmd.Add(env, args...)
}

// Add creates an environment variable for the given argument and adds that environment variable to the given commandline.
// If the given argument is short or it contains a '$', it is added directly.
func (c *Cmdline) Add(env map[string]string, args ...string) *Cmdline {
	for _, arg := range args {
		if (len(arg) <= shortArgLen) || strings.Contains(arg, "$") || env == nil {
			c.cmd = append(c.cmd, arg)
		} else {
			// Look for an existing environment variable with the same value
			key := ""
			for k, v := range env {
				if v == arg {
					key = k
					break
				}
			}
			if key == "" {
				// No environment variable with same value found, create a new one
				key = fmt.Sprintf("A%02d", len(env))
				env[key] = arg
			}
			// Add to commandline
			c.cmd = append(c.cmd, fmt.Sprintf("$%s", key))
		}
	}
	return c
}

func (c *Cmdline) String() string {
	return strings.Join(c.cmd, " ")
}
