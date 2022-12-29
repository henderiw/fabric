package fabric

import (
	"fmt"
	"strings"

	topov1alpha1 "github.com/henderiw-k8s-lcnc/topology/apis/topo/v1alpha1"
	"gonum.org/v1/gonum/graph"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Link interface {
	From() graph.Node
	To() graph.Node
	ReversedLine() graph.Line
	ID() int64

	GetKRMLink() *topov1alpha1.Link
}

type linkInfo struct {
	from      Node
	to        Node
	fromItfce string
	toItfce   string
}

func NewLink(oi *originInfo, li *linkInfo, lID int64) Link {
	return &link{
		F:       li.from,
		T:       li.to,
		UID:     lID,
		linkKRM: li.buildLink(oi),
	}
}

type link struct {
	F, T    graph.Node
	UID     int64
	linkKRM *topov1alpha1.Link
}

func (l *link) From() graph.Node         { return l.F }
func (l *link) To() graph.Node           { return l.T }
func (l *link) ReversedLine() graph.Line { l.F, l.T = l.T, l.F; return l }
func (l *link) ID() int64                { return l.UID }
func (l *link) GetKRMLink() *topov1alpha1.Link {
	return l.linkKRM
}

func (l *linkInfo) buildLink(oi *originInfo) *topov1alpha1.Link {
	labels := map[string]string{
		//LabelKeyOrganization:     cr.GetOrganization(),
		//LabelKeyDeployment:       cr.GetDeployment(),
		//LabelKeyAvailabilityZone: cr.GetAvailabilityZone(),
		topov1alpha1.LabelKeyTopology: oi.name,
	}

	return &topov1alpha1.Link{
		TypeMeta: metav1.TypeMeta{
			APIVersion: topov1alpha1.GroupVersion.String(),
			Kind:       topov1alpha1.LinkKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.Join([]string{oi.name, l.getName()}, "."),
			Namespace: oi.namespace,
			Labels:    labels,
		},
		Spec: topov1alpha1.LinkSpec{
			Properties: &topov1alpha1.LinkProperties{
				Kind: topov1alpha1.LinkKindInfra,
				Endpoints: []*topov1alpha1.Endpoints{
					{
						InterfaceName: l.fromItfce,
						NodeName:      l.from.GetKRMNode().GetName(),
						Kind:          topov1alpha1.EndpointKindInfra,
					},
					{
						InterfaceName: l.toItfce,
						NodeName:      l.to.GetKRMNode().GetName(),
						Kind:          topov1alpha1.EndpointKindInfra,
					},
				},
			},
		},
	}
}

func (l *linkInfo) getName() string {
	linkName := fmt.Sprintf("%s-%s-%s-%s",
		l.from.String(),
		l.fromItfce,
		l.to.String(),
		l.toItfce)
	return strings.ReplaceAll(linkName, "/", "-")
}
