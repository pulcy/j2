package kubernetes

import (
	"strings"

	k8s "github.com/YakLabs/k8s-client"
	"github.com/juju/errgo"
	"github.com/pulcy/j2/jobs"
	pkg "github.com/pulcy/j2/pkg/kubernetes"
)

const (
	NodeSelectorOpIn           = "In"
	NodeSelectorOpNotIn        = "NotIn"
	NodeSelectorOpExists       = "Exists"
	NodeSelectorOpDoesNotExist = "DoesNotExist"
	NodeSelectorOpGt           = "Gt"
	NodeSelectorOpLt           = "Lt"
)

// createAffinity creates an affinity object for the given constraints.
func createAffinity(constraints jobs.Constraints, tg *jobs.TaskGroup, pod pod, ctx generatorContext) (*k8s.Affinity, error) {
	if constraints.Len() == 0 {
		return nil, nil
	}

	a := &k8s.Affinity{}
	nodeSelector := k8s.NodeSelectorTerm{}
	podSelector := k8s.PodAffinityTerm{}
	antiPodSelector := k8s.PodAffinityTerm{}

	for _, c := range constraints {
		if strings.HasPrefix(c.Attribute, jobs.MetaAttributePrefix) {
			// meta.<somekey>
			key := c.Attribute[len(jobs.MetaAttributePrefix):]
			req := k8s.NodeSelectorRequirement{
				Key: key,
			}
			if c.OperatorEquals(jobs.OperatorEqual) {
				req.Operator = NodeSelectorOpIn
				req.Values = []string{c.Value}
			} else if c.OperatorEquals(jobs.OperatorNotEqual) {
				req.Operator = NodeSelectorOpNotIn
				req.Values = []string{c.Value}
			} else {
				return nil, errgo.WithCausef(nil, ValidationError, "constraint with attribute '%s' has unsupported operator '%s'", c.Attribute, c.Operator)
			}
			nodeSelector.MatchExpressions = append(nodeSelector.MatchExpressions, req)
		} else {
			switch c.Attribute {
			case jobs.AttributeNodeID:
				req := k8s.NodeSelectorRequirement{
					Key: "id",
				}
				if c.OperatorEquals(jobs.OperatorEqual) {
					req.Operator = NodeSelectorOpIn
					req.Values = []string{c.Value}
				} else if c.OperatorEquals(jobs.OperatorNotEqual) {
					req.Operator = NodeSelectorOpNotIn
					req.Values = []string{c.Value}
				} else {
					return nil, errgo.WithCausef(nil, ValidationError, "constraint with attribute '%s' has unsupported operator '%s'", c.Attribute, c.Operator)
				}
				nodeSelector.MatchExpressions = append(nodeSelector.MatchExpressions, req)
			case jobs.AttributeTaskGroup:
				name := jobs.TaskGroupName(c.Value)
				if err := name.Validate(); err != nil {
					return nil, maskAny(err)
				}
				group, err := tg.TaskGroup(name)
				if err != nil {
					return nil, maskAny(err)
				}
				var term *k8s.PodAffinityTerm
				if c.OperatorEquals(jobs.OperatorEqual) {
					term = &podSelector
				} else if c.OperatorEquals(jobs.OperatorNotEqual) {
					term = &antiPodSelector
				} else {
					return nil, errgo.WithCausef(nil, ValidationError, "constraint with attribute '%s' has unsupported operator '%s'", c.Attribute, c.Operator)
				}
				if term.LabelSelector == nil {
					term.LabelSelector = newLabelSelector()
				}
				term.TopologyKey = "node"
				term.LabelSelector.MatchLabels[pkg.LabelTaskGroupFullName] = pkg.ResourceName(group.FullName())
			default:
				return nil, errgo.WithCausef(nil, ValidationError, "Unknown constraint attribute '%s'", c.Attribute)
			}
		}
	}

	if len(nodeSelector.MatchExpressions) > 0 {
		a.NodeAffinity = &k8s.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &k8s.NodeSelector{
				NodeSelectorTerms: []k8s.NodeSelectorTerm{nodeSelector},
			},
		}
	}
	if podSelector.LabelSelector != nil {
		a.PodAffinity = &k8s.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []k8s.PodAffinityTerm{podSelector},
		}
	}
	if antiPodSelector.LabelSelector != nil {
		a.PodAntiAffinity = &k8s.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []k8s.PodAffinityTerm{antiPodSelector},
		}
	}

	return a, nil
}

func newLabelSelector() *k8s.LabelSelector {
	return &k8s.LabelSelector{
		MatchLabels: map[string]string{},
	}
}
