package fabric

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	targetv1 "github.com/henderiw-k8s-lcnc/target/apis/target/v1"
	topov1alpha1 "github.com/henderiw-k8s-lcnc/topology/apis/topo/v1alpha1"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/multi"
	"k8s.io/apimachinery/pkg/labels"
)

type Fabric interface {
	GetNodes() []*topov1alpha1.Node
	GetLinks() []*topov1alpha1.Link
	PrintGraph()
	GenerateJsonFile() error
	Print()
}

type Config struct {
	Name             string
	Namespace        string
	Location         *topov1alpha1.Location
	Organization     string
	Deployment       string
	AvailabilityZone string
	MasterTemplates  []*topov1alpha1.Template
	ChildTemplates   []*topov1alpha1.Template
}

type fabric struct {
	graph *multi.UndirectedGraph
	cfg   *Config
}

func New(c *Config) (Fabric, error) {
	r := &fabric{
		graph: multi.NewUndirectedGraph(),
		cfg:   c,
	}

	if err := r.validateTemplates(); err != nil {
		return nil, err
	}

	// a template can have multiple template references so we need to
	// build one fabric topology based on all the references
	t, err := r.buildNewFabricTemplate()
	if err != nil {
		return nil, err
	}

	// populateNodes processes the template and populates
	// the leaf, spine, superspines and borderleafs in the graph
	if err := r.populateNodes(t); err != nil {
		return nil, err
	}

	// connect interconnects the nodes in the graph using the template
	// connect spine to leaf
	// connect spine to superspine
	// connect spine to borderleaf
	if err := r.connect(t); err != nil {
		return nil, err
	}

	return r, nil
}

func (f *fabric) GetNodes() []*topov1alpha1.Node {
	nodes := make([]*topov1alpha1.Node, 0)
	it := f.graph.Nodes()
	if it == nil {
		return nodes
	}

	for it.Next() {
		n := it.Node().(Node)
		nodes = append(nodes, n.GetKRMNode())
	}
	return nodes
}

func (f *fabric) getNodes() []Node {
	nodes := make([]Node, 0)
	it := f.graph.Nodes()
	if it == nil {
		return nodes
	}

	for it.Next() {
		n := it.Node().(Node)
		nodes = append(nodes, n)
	}
	return nodes
}

func (f *fabric) GetLinks() []*topov1alpha1.Link {
	links := make([]*topov1alpha1.Link, 0)
	it := f.graph.Edges()
	if it == nil {
		return links
	}

	for it.Next() {
		edge := it.Edge().(multi.Edge)
		for edge.Lines.Next() {
			l := edge.Lines.Line().(Link)
			links = append(links, l.GetKRMLink())
		}
	}
	return links
}

func (f *fabric) getLinks() []Link {
	links := make([]Link, 0)
	it := f.graph.Edges()
	if it == nil {
		return links
	}

	for it.Next() {
		edge := it.Edge().(multi.Edge)
		for edge.Lines.Next() {
			l := edge.Lines.Line().(Link)
			links = append(links, l)
		}
	}
	return links
}

func (r *fabric) addNode(n Node) {
	r.graph.AddNode(n)
}

func (f *fabric) nodesByLabel(selector labels.Selector) (nodes []Node) {
	it := f.graph.Nodes()
	if it == nil {
		return nil
	}
	n := it.Len()
	switch {
	case n == 0:
		return nil
	case n < 0:
		n = 0
	}
	for it.Next() {
		node := it.Node().(Node)
		if selector.Matches(node.GetLabels()) {
			nodes = append(nodes, node)
		}
	}
	if len(nodes) == 0 {
		return nil
	}
	return nodes
}

func (f *fabric) addLink(oi *originInfo, li *linkInfo) Link {
	l := f.graph.NewLine(li.from, li.to)
	return NewLink(oi, li, l.ID())
}

func (f *fabric) PrintGraph() {
	result, _ := dot.Marshal(f.graph, "", "", "  ")
	fmt.Print(string(result))
}

type TopologyJsonNode struct {
	ID    int                   `json:"id"`
	Label string                `json:"label"`
	Level int                   `json:"level"`
	Nos   string                `json:"nos,omitempty"`
	Cid   string                `json:"cid"`
	Data  *TopologyJsonNodedata `json:"data,omitempty"`
}

type TopologyJsonNodedata struct {
	ExpectedSWVersion string `json:"expectedSWVersion,omitempty"`
	MgmtIP            string `json:"mgmtIp,omitempty"`
	Model             string `json:"model,omitempty"`
}

type TopologyJsonLink struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type TopologyJsonFile struct {
	Nodes []*TopologyJsonNode `json:"nodes,omitempty"`
	Edges []*TopologyJsonLink `json:"edges,omitempty"`
}

func (f *fabric) GenerateJsonFile() error {
	t := &TopologyJsonFile{
		Nodes: []*TopologyJsonNode{},
		Edges: []*TopologyJsonLink{},
	}

	nodes := f.getNodes()
	for _, n := range nodes {

		vendorType := ""
		switch n.GetKRMNode().Spec.Properties.VendorType {
		case targetv1.VendorTypeNokiaSRL:
			vendorType = "srlinux"
		case targetv1.VendorTypeNokiaSROS:
			vendorType = "sros"
		default:
			vendorType = string(n.GetKRMNode().Spec.Properties.VendorType)
		}

		t.Nodes = append(t.Nodes, &TopologyJsonNode{
			ID:    int(n.ID()),
			Level: topov1alpha1.GetLevel(topov1alpha1.Position(n.GetKRMNode().Spec.Properties.Position)),
			Label: n.String(),
			Nos:   vendorType,
			Cid:   string(n.GetKRMNode().Spec.Properties.Position),
			Data: &TopologyJsonNodedata{
				Model: n.GetKRMNode().Spec.Properties.Platform,
			},
		})
	}

	links := f.getLinks()
	for _, l := range links {
		t.Edges = append(t.Edges, &TopologyJsonLink{
			From: int(l.From().(Node).ID()),
			To:   int(l.To().(Node).ID()),
		})
	}

	j, err := json.MarshalIndent(t, "", "\t")
	if err != nil {
		return err
	}

	fmt.Printf("json output: \n%s\n", j)

	filepath := filepath.Join("out", "fabric.json")

	file, err := os.Create(filepath)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath, j, 0644); err != nil {
		return err
	}

	defer file.Close()
	return nil
}

func (r *fabric) Print() {
	for _, n := range r.GetNodes() {
		b, err := json.MarshalIndent(n, "", "  ")
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(string(b))
	}
	for _, l := range r.GetLinks() {
		b, err := json.MarshalIndent(l, "", "  ")
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(string(b))
	}
}
