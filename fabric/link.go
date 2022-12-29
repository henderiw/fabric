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
	//String() string

	//FromNodeName() string
	//ToNodeName() string
	//FromIfName() string
	//ToIfName() string

	GetKRMLink() *topov1alpha1.Link

	//Attributes() []encoding.Attribute
	//SetLabel(label map[string]string) error
	//UpdateLabel(label map[string]string) error
	//GetLabels() labels.Set
}

type linkInfo struct {
	from      Node
	to        Node
	fromItfce string
	toItfce   string
}

func NewLink(oi *originInfo, li *linkInfo, lID int64) Link {
	return &link{
		F:   li.from,
		T:   li.to,
		UID: lID,
		//attrs: labels.Set(map[string]string{}),
		linkKRM: buildLink(oi, li),
	}
}

type link struct {
	F, T graph.Node
	UID  int64
	//attrs labels.Set
	linkKRM *topov1alpha1.Link
}

func (l *link) From() graph.Node         { return l.F }
func (l *link) To() graph.Node           { return l.T }
func (l *link) ReversedLine() graph.Line { l.F, l.T = l.T, l.F; return l }
func (l *link) ID() int64                { return l.UID }

func (l *link) String() string {
	from := l.From().(Node)
	to := l.To().(Node)
	linkName := fmt.Sprintf("%s-%s-%s-%s", from.String(), l.GetKRMLink().GetLabels()[from.String()], to.String(), l.GetKRMLink().GetLabels()[to.String()])
	return strings.ReplaceAll(linkName, "/", "-")
}

func (l *link) GetKRMLink() *topov1alpha1.Link {
	return l.linkKRM
}

//func (l *link) FromNodeName() string { return l.From().(Node).String() }
//func (l *link) ToNodeName() string   { return l.To().(Node).String() }
//func (l *link) FromIfName() string   { return l.GetKRMLink().GetLabels()[l.FromNodeName()] }
//func (l *link) ToIfName() string     { return l.GetKRMLink().GetLabels()[l.ToNodeName()] }

// Attributes implements the encoding.Attributer interface.
/*
func (l *link) Attributes() []encoding.Attribute {
	var keys []string
	for key := range l.attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var attrs []encoding.Attribute
	for _, key := range keys {
		attr := encoding.Attribute{Key: key, Value: l.attrs[key]}
		attrs = append(attrs, attr)
	}
	return attrs
}
func (l *link) SetLabel(label map[string]string) error {
	l.attrs = labels.Set(label)
	return nil
}
func (l *link) UpdateLabel(label map[string]string) error {
	l.attrs = labels.Merge(labels.Set(label), l.attrs)
	return nil
}

func (l *link) GetLabels() labels.Set { return l.attrs }
*/

func buildLink(oi *originInfo, li *linkInfo) *topov1alpha1.Link {
	labels := map[string]string{
		//LabelKeyOrganization:     cr.GetOrganization(),
		//LabelKeyDeployment:       cr.GetDeployment(),
		//LabelKeyAvailabilityZone: cr.GetAvailabilityZone(),
		topov1alpha1.LabelKeyTopology: oi.name,
	}

	return &topov1alpha1.Link{
		TypeMeta: metav1.TypeMeta{
			APIVersion: topov1alpha1.GroupVersion.String(),
			Kind: topov1alpha1.LinkKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.Join([]string{oi.name, li.getName()}, "."),
			Namespace: oi.namespace,
			Labels:    labels,
		},
		Spec: topov1alpha1.LinkSpec{
			Properties: &topov1alpha1.LinkProperties{
				Kind: topov1alpha1.LinkKindInfra,
				Endpoints: []*topov1alpha1.Endpoints{
					{
						InterfaceName: li.fromItfce,
						NodeName:      li.from.String(),
						Kind:          topov1alpha1.EndpointKindInfra,
					},
					{
						InterfaceName: li.toItfce,
						NodeName:      li.to.String(),
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
