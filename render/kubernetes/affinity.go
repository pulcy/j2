package kubernetes

import (
	"strings"

	"github.com/juju/errgo"
	"github.com/pulcy/j2/jobs"
	k8s "github.com/pulcy/j2/pkg/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	metav1 "k8s.io/client-go/pkg/apis/meta/v1"
)

// createAffinity creates an affinity object for the given constraints.
func createAffinity(constraints jobs.Constraints, tg *jobs.TaskGroup, pod pod, ctx generatorContext) (*v1.Affinity, error) {
	if constraints.Len() == 0 {
		return nil, nil
	}

	a := &v1.Affinity{}
	nodeSelector := v1.NodeSelectorTerm{}
	podSelector := v1.PodAffinityTerm{}
	antiPodSelector := v1.PodAffinityTerm{}

	for _, c := range constraints {
		if strings.HasPrefix(c.Attribute, jobs.MetaAttributePrefix) {
			// meta.<somekey>
			key := c.Attribute[len(jobs.MetaAttributePrefix):]
			req := v1.NodeSelectorRequirement{
				Key: key,
			}
			if c.OperatorEquals(jobs.OperatorEqual) {
				req.Operator = v1.NodeSelectorOpIn
				req.Values = []string{c.Value}
			} else if c.OperatorEquals(jobs.OperatorNotEqual) {
				req.Operator = v1.NodeSelectorOpNotIn
				req.Values = []string{c.Value}
			} else {
				return nil, errgo.WithCausef(nil, ValidationError, "constraint with attribute '%s' has unsupported operator '%s'", c.Attribute, c.Operator)
			}
			nodeSelector.MatchExpressions = append(nodeSelector.MatchExpressions, req)
		} else {
			switch c.Attribute {
			case jobs.AttributeNodeID:
				req := v1.NodeSelectorRequirement{
					Key: "id",
				}
				if c.OperatorEquals(jobs.OperatorEqual) {
					req.Operator = v1.NodeSelectorOpIn
					req.Values = []string{c.Value}
				} else if c.OperatorEquals(jobs.OperatorNotEqual) {
					req.Operator = v1.NodeSelectorOpNotIn
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
				var term *v1.PodAffinityTerm
				if c.OperatorEquals(jobs.OperatorEqual) {
					term = &podSelector
				} else if c.OperatorEquals(jobs.OperatorNotEqual) {
					term = &antiPodSelector
				} else {
					return nil, errgo.WithCausef(nil, ValidationError, "constraint with attribute '%s' has unsupported operator '%s'", c.Attribute, c.Operator)
				}
				if term.LabelSelector == nil {
					term.LabelSelector = &metav1.LabelSelector{}
				}
				term.TopologyKey = "node"
				term.LabelSelector.MatchLabels[k8s.LabelTaskGroupFullName] = group.FullName()
			default:
				return nil, errgo.WithCausef(nil, ValidationError, "Unknown constraint attribute '%s'", c.Attribute)
			}
		}
	}

	if len(nodeSelector.MatchExpressions) > 0 {
		a.NodeAffinity = &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: []v1.NodeSelectorTerm{nodeSelector},
			},
		}
	}
	if podSelector.LabelSelector != nil {
		a.PodAffinity = &v1.PodAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{podSelector},
		}
	}
	if antiPodSelector.LabelSelector != nil {
		a.PodAntiAffinity = &v1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{antiPodSelector},
		}
	}

	return a, nil
}
