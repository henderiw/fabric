package fabric

import (
	"fmt"
	"sort"
	"strings"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/encoding"
	"k8s.io/apimachinery/pkg/labels"
)

type Link interface {
	From() graph.Node
	To() graph.Node
	ReversedLine() graph.Line
	ID() int64
	String() string

	FromNodeName() string
	ToNodeName() string
	FromIfName() string
	ToIfName() string

	Attributes() []encoding.Attribute
	SetLabel(label map[string]string) error
	UpdateLabel(label map[string]string) error
	GetLabels() labels.Set
}

func NewLink(l graph.Line) Link {
	return &link{
		F:     l.From(),
		T:     l.To(),
		UID:   l.ID(),
		attrs: labels.Set(map[string]string{}),
	}
}

type link struct {
	F, T  graph.Node
	UID   int64
	attrs labels.Set
}

func (l *link) From() graph.Node         { return l.F }
func (l *link) To() graph.Node           { return l.T }
func (l *link) ReversedLine() graph.Line { l.F, l.T = l.T, l.F; return l }
func (l *link) ID() int64                { return l.UID }

func (l *link) String() string {
	from := l.From().(Node)
	to := l.To().(Node)
	linkName := fmt.Sprintf("%s-%s-%s-%s", from.String(), l.GetLabels()[from.String()], to.String(), l.GetLabels()[to.String()])
	return strings.ReplaceAll(linkName, "/", "-")
}

func (l *link) FromNodeName() string { return l.From().(Node).String() }
func (l *link) ToNodeName() string   { return l.To().(Node).String() }
func (l *link) FromIfName() string   { return l.GetLabels()[l.FromNodeName()] }
func (l *link) ToIfName() string     { return l.GetLabels()[l.ToNodeName()] }

// Attributes implements the encoding.Attributer interface.
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
