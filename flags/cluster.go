package flags

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/juju/errgo"
	"github.com/mitchellh/mapstructure"
)

const (
	defaultInstanceCount = 3
)

// Cluster contains all variables describing a cluster (deployment target)
type Cluster struct {
	// Unique (not so easy to guess) ID
	ID string `mapstructure:"id"`
	// Name within the domain e.g. alpha-c32
	Stack string `mapstructure:"stack"`
	// Domain name e.g. pulcy.com
	Domain string `mapstructure:"domain"`
	// SSH tunnel needed to reach the cluster (optional)
	Tunnel string `mapstructure:"tunnel,omitempty"`
	// Size of the cluster (in instances==machines)
	InstanceCount int `mapstructure:"instance-count,omitempty"`

	DefaultOptions Options `mapstructure:"default-options,omitempty"`
}

// validate checks the values in the given cluster
func (c Cluster) validate() error {
	if c.ID == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "ID missing"))
	}
	if c.Stack == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "Stack missing"))
	}
	if c.Domain == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "Domain missing"))
	}
	if c.Tunnel == "" {
		return maskAny(errgo.WithCausef(nil, ValidationError, "Tunnel missing"))
	}
	if c.InstanceCount == 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "InstanceCount missing"))
	} else if c.InstanceCount < 0 {
		return maskAny(errgo.WithCausef(nil, ValidationError, "InstanceCount negative"))
	}
	return nil
}

func (c *Cluster) setDefaults() {
	if c.Tunnel == "" {
		c.Tunnel = fmt.Sprintf("%s.%s", c.Stack, c.Domain)
	}
	if c.InstanceCount == 0 {
		c.InstanceCount = defaultInstanceCount
	}
}

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

func (c *Cluster) parse(list *ast.ObjectList) error {
	list = list.Children()
	if len(list.Items) != 1 {
		return errgo.New("only one 'cluster' block allowed")
	}
	obj := list.Items[0]
	ot, ok := obj.Val.(*ast.ObjectType)
	if !ok {
		return errgo.New("cluster is expected to be an ObjectType")
	}
	c.Stack = obj.Keys[0].Token.Value().(string)

	// Decode the full thing into a map[string]interface for ease
	var m map[string]interface{}
	if err := hcl.DecodeObject(&m, obj.Val); err != nil {
		return maskAny(err)
	}
	delete(m, "default-options")

	// Decode the rest
	if err := mapstructure.WeakDecode(m, c); err != nil {
		return maskAny(err)
	}

	if o := ot.List.Filter("default-options"); len(o.Items) > 0 {
		for _, o := range o.Elem().Items {
			var m map[string]string
			if err := hcl.DecodeObject(&m, o.Val); err != nil {
				return maskAny(err)
			}
			// Merge key/value pairs into myself
			for k, v := range m {
				c.DefaultOptions.set(k, v)
			}
		}
	}

	return nil
}
