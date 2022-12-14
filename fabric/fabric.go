package fabric

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/yndd/ndd-runtime/pkg/logging"
	targetv1 "github.com/yndd/target/apis/target/v1"
	topov1alpha1 "github.com/yndd/topology/apis/topo/v1alpha1"
	"gonum.org/v1/gonum/graph/encoding/dot"
	"gonum.org/v1/gonum/graph/multi"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Option can be used to manipulate Fabric config.
type Option func(Fabric)

// WithLogger specifies how the Fabric logs messages.
func WithLogger(log logging.Logger) Option {
	return func(f Fabric) {
		f.SetLogger(log)
	}
}

// WithClient specifies the fabric to use within the client.
func WithClient(c client.Client) Option {
	return func(f Fabric) {
		f.SetClient(c)
	}
}

// WithClient specifies the fabric to use within the client.
func WithLocation(l *topov1alpha1.Location) Option {
	return func(f Fabric) {
		f.SetLocation(l)
	}
}

type Fabric interface {
	GetNodes() []Node
	GetLinks() []Link
	PrintNodes()
	PrintLinks()
	PrintGraph()
	GenerateJsonFile() error

	SetLogger(logger logging.Logger)
	SetClient(c client.Client)
	SetLocation(l *topov1alpha1.Location)
}

func New(t *topov1alpha1.Template, opts ...Option) (Fabric, error) {
	f := &fabric{
		graph:     multi.NewUndirectedGraph(),
		namespace: t.Namespace,
	}

	for _, opt := range opts {
		opt(f)
	}

	// a template can have multiple template/definition references so we need to parse them
	// to build one fabric topology
	newt, err := f.parseTemplate(t.Spec.Properties.Fabric)
	if err != nil {
		return nil, err
	}

	// process leaf/spine nodes
	// p is number of pod definitions
	for p, pod := range newt.Pod {
		// i is the number of pods in a definition
		for i := uint32(0); i < pod.GetPodNumber(); i++ {
			// podIndex is pod template index * pod index within the template
			podIndex := (uint32(p) + 1) * (i + 1)

			//log.Debug("podIndex", "podIndex", podIndex)

			// tier 2 -> spines in the pod
			if err := f.processTier(topov1alpha1.PositionSpine, podIndex, pod.Tier2, pod.IsToBeDeployed()); err != nil {
				return nil, err
			}
			// tier 3 -> leafs in the pod
			if err := f.processTier(topov1alpha1.PositionLeaf, podIndex, pod.Tier3, pod.IsToBeDeployed()); err != nil {
				return nil, err
			}
		}
	}

	// proces superspines
	// the superspine is equal to the amount of spines per pod and multiplied with the number in the template
	if newt.Tier1 != nil {
		// process superspine nodes
		for n := uint32(0); n < newt.GetSuperSpines(); n++ {
			if err := f.processTier(topov1alpha1.PositionSuperspine, n+1, newt.Tier1, true); err != nil {
				return nil, err
			}
		}
	}

	// process borderleafs
	if newt.BorderLeaf != nil {
		// process borderleafs nodes
		if err := f.processTier(topov1alpha1.PositionBorderLeaf, 1, newt.BorderLeaf, true); err != nil {
			return nil, err
		}

	}

	// wire things

	// process spine-leaf links
	for p, pod := range newt.Pod {
		// i is the number of pods in a definition
		for i := 0; i < int(pod.GetPodNumber()); i++ {
			// podIndex is pod template index * pod index within the template
			podIndex := (p + 1) * (i + 1)

			// identify all the leafs and spines in the podIndex
			// from -> tier2 or spines
			// to -> tier 3 or leafs
			tier2Selector := labels.NewSelector()
			tier3Selector := labels.NewSelector()

			tier2Req, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionSpine)})
			tier3Req, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionLeaf)})

			// select the POD Index
			podIdxReq, _ := labels.NewRequirement(KeyPodIndex, selection.Equals, []string{strconv.Itoa(podIndex)})

			tier2Selector = tier2Selector.Add(*tier2Req, *podIdxReq)
			tier3Selector = tier3Selector.Add(*tier3Req, *podIdxReq)

			tier2Nodes := f.nodesByLabel(tier2Selector)
			tier3Nodes := f.nodesByLabel(tier3Selector)

			for _, tier2Node := range tier2Nodes {
				//tier2NodeIndex := uint32(n) + 1
				for _, tier3Node := range tier3Nodes {
					//tier3NodeIndex := uint32(m) + 1
					// validate if the uplinks per node is not greater than max uplinks
					// otherwise there is a conflict and the algorithm behind will create
					// overlapping indexes
					uplinksPerNode := tier3Node.GetUplinkPerNode()
					if uplinksPerNode > newt.Settings.MaxUplinksTier3ToTier2 {
						return nil, fmt.Errorf("uplink per node %d can not be bigger than maxUplinksTier3ToTier2 %d",
							uplinksPerNode, newt.Settings.MaxUplinksTier3ToTier2)
					}

					// the algorithm needs to avoid reindixing if changes happen -> introduced maxNumUplinks
					// the allocation is first allocating the uplink Index
					// u represnts the actual uplink index
					// spine Index    -> actualUplinkId + (actual leafs  * max uplinks)
					// leaf  Index    -> actualUplinkId + (actual spines * max uplinks)
					// actualUplinkId = u + 1 -> counting starts at 1
					// actual leafs   = tier3NodeIndex - 1 -> counting from 0
					// actual spines  = tier2NodeIndex - 1 -> counting from 0
					// max uplinks    = mergedTemplate.MaxUplinksTier3ToTier2
					for u := uint32(0); u < uplinksPerNode; u++ {

						l := f.addLink(tier2Node, tier3Node)

						tier3NodeIndex, err := strconv.Atoi(tier3Node.GetRelativeNodeIndex())
						if err != nil {
							return nil, err
						}
						tier2NodeIndex, err := strconv.Atoi(tier2Node.GetRelativeNodeIndex())
						if err != nil {
							return nil, err
						}

						label := map[string]string{
							tier2Node.String(): tier2Node.GetInterfaceName(u + 1 + ((uint32(tier3NodeIndex) - 1) * newt.Settings.MaxUplinksTier3ToTier2)),
							tier3Node.String(): tier3Node.GetInterfaceNameWithPlatfromOffset(u + 1 + ((uint32(tier2NodeIndex) - 1) * newt.Settings.MaxUplinksTier3ToTier2)),
						}
						l.SetLabel(label)

						f.graph.SetLine(l)

						f.log.Debug("Adding link", "from:", tier2Node.String(), "itfce", label[tier2Node.String()], "to:", tier3Node.String(), "itfce", label[tier3Node.String()])
					}
				}
			}
		}

		tier1Selector := labels.NewSelector()
		tier1Req, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionSuperspine)})
		tier1Selector = tier1Selector.Add(*tier1Req)
		tier1Nodes := f.nodesByLabel(tier1Selector)

		tier2Selector := labels.NewSelector()
		tier2Req, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionSpine)})
		tier2Selector = tier2Selector.Add(*tier2Req)
		tier2Nodes := f.nodesByLabel(tier2Selector)

		// process superspine-spine links
		for _, tier1Node := range tier1Nodes {
			for _, tier2Node := range tier2Nodes {
				// validate if the uplinks per node is not greater than max uplinks
				// otherwise there is a conflict and the algorithm behind will create
				// overlapping indexes
				uplinksPerNode := tier2Node.GetUplinkPerNode()
				if uplinksPerNode > newt.Settings.MaxUplinksTier2ToTier1 {
					return nil, fmt.Errorf("uplink per node %d can not be bigger than maxUplinksTier2ToTier1 %d", uplinksPerNode, newt.Settings.MaxUplinksTier2ToTier1)
				}

				// spine and superspine line up so we only create a link if there is a match
				// on the indexes
				if tier2Node.GetRelativeNodeIndex() == tier1Node.GetPlaneIndex() {
					// the algorithm needs to avoid reindixing if changes happen -> introduced maxNumUplinks
					// the allocation is first allocating the uplink Index
					// u represnts the actual uplink index
					// superspine Index -> actualUplinkId + (actual podIndex  * max uplinks)
					// spine Index      -> actualUplinkId + (actual spines per plane * max uplinks)
					// actualUplinkId          = u + 1 -> counting starts at 1
					// actual PodIndex         = p +1
					// actual spines per plane = tier1Node.GetNodePlaneIndex() - 1
					// max uplinks             = mergedTemplate.MaxUplinksTier2ToTier1
					for u := uint32(0); u < uplinksPerNode; u++ {

						l := f.addLink(tier1Node, tier2Node)

						podIndex, err := strconv.Atoi(tier2Node.GetPodIndex())
						if err != nil {
							return nil, err
						}
						relativeIndex, err := strconv.Atoi(tier1Node.GetRelativeNodeIndex())
						if err != nil {
							return nil, err
						}

						label := map[string]string{
							tier1Node.String(): tier1Node.GetInterfaceName(u + 1 + (uint32(podIndex-1) * newt.Settings.MaxUplinksTier2ToTier1)),
							tier2Node.String(): tier2Node.GetInterfaceNameWithPlatfromOffset(u + 1 + (uint32(relativeIndex-1) * newt.Settings.MaxUplinksTier2ToTier1)),
						}
						l.SetLabel(label)

						f.graph.SetLine(l)

						f.log.Debug("Adding link", "from:", tier1Node.String(), "itfce", label[tier1Node.String()], "to:", tier2Node.String(), "itfce", label[tier2Node.String()])
					}
				}
			}
		}

		blSelector := labels.NewSelector()
		blReq, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionBorderLeaf)})
		blSelector = blSelector.Add(*blReq)
		blNodes := f.nodesByLabel(blSelector)

		// process borderleaf-spine links
		for _, blNode := range blNodes {
			for _, tier2Node := range tier2Nodes {
				// validate if the uplinks per node is not greater than max uplinks
				// otherwise there is a conflict and the algorithm behind will create
				// overlapping indexes
				uplinksPerNode := tier2Node.GetUplinkPerNode()
				if uplinksPerNode > newt.Settings.MaxUplinksTier2ToTier1 {
					return nil, fmt.Errorf("uplink per node %d can not be bigger than maxUplinksTier2ToTier1 %d", uplinksPerNode, newt.Settings.MaxUplinksTier2ToTier1)
				}

				for u := uint32(0); u < uplinksPerNode; u++ {

					l := f.addLink(blNode, tier2Node)

					podIndex, err := strconv.Atoi(tier2Node.GetPodIndex())
					if err != nil {
						return nil, err
					}
					if uint32(podIndex) > newt.Settings.MaxUplinksTier2ToTier1 {
						return nil, fmt.Errorf("spines per pod cannot be bigger than maxSpinesPerPod")
					}
					tier2NodeIndex, err := strconv.Atoi(tier2Node.GetRelativeNodeIndex())
					if err != nil {
						return nil, err
					}
					blNodeIndex, err := strconv.Atoi(blNode.GetRelativeNodeIndex())
					if err != nil {
						return nil, err
					}

					label := map[string]string{
						blNode.String():    blNode.GetInterfaceName(u + 1 + ((uint32(podIndex-1) + ((uint32(tier2NodeIndex) - 1) * newt.Settings.MaxSpinesPerPod)) * newt.Settings.MaxUplinksTier2ToTier1)),
						tier2Node.String(): tier2Node.GetInterfaceNameWithPlatfromOffset(u + 1 + (uint32(blNodeIndex-1) * newt.Settings.MaxUplinksTier2ToTier1)),
					}
					l.SetLabel(label)

					f.graph.SetLine(l)

					f.log.Debug("Adding link", "from:", blNode.String(), "itfce", label[blNode.String()], "to:", tier2Node.String(), "itfce", label[tier2Node.String()])
				}
			}
		}
	}

	return f, nil
}

type fabric struct {
	log       logging.Logger
	client    client.Client
	graph     *multi.UndirectedGraph
	location  *topov1alpha1.Location
	namespace string
}

func (f *fabric) SetLogger(log logging.Logger)         { f.log = log }
func (f *fabric) SetClient(c client.Client)            { f.client = c }
func (f *fabric) SetLocation(l *topov1alpha1.Location) { f.location = l }

func (f *fabric) GetNodes() []Node {
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

func (f *fabric) GetLinks() []Link {
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

func (f *fabric) PrintNodes() {
	nodes := f.GetNodes()
	for _, n := range nodes {
		f.printNode(n)
	}
}

func (f *fabric) printNode(n Node) {
	if n.GetPosition() == string(topov1alpha1.PositionSuperspine) {
		f.log.Debug("node",
			"id", n.ID(),
			"position", n.GetPosition(),
			"nodeName", n.String(),
			"planeIndex", n.GetPlaneIndex(),
			"relativeNodeIndex", n.GetRelativeNodeIndex(),
			//"vendorType", n.GetVendorType(),
			//"platform", n.GetPlatform(),
			"location", n.GetLocation(),
		)
	} else {
		f.log.Debug("node",
			"id", n.ID(),
			"position", n.GetPosition(),
			"nodeName", n.String(),
			"podIndex", n.GetPodIndex(),
			"relativeNodeIndex", n.GetRelativeNodeIndex(),
			//"vendorType", n.GetVendorType(),
			//"platform", n.GetPlatform(),
			"location", n.GetLocation(),
		)
	}
}

func (f *fabric) PrintLinks() {
	for _, l := range f.GetLinks() {
		f.printLink(l)
	}
}

func (f *fabric) printLink(l Link) {
	from := l.From().(Node)
	to := l.To().(Node)

	if from.GetPosition() == string(topov1alpha1.PositionSuperspine) {
		f.log.Debug("link",
			"from nodeName", from.String(),
			"from planeIndex", from.GetPlaneIndex(),
			"from ifName", l.GetLabels()[from.String()],
			"to nodeName", to.String(),
			"to podIndex", to.GetPodIndex(),
			"to ifName", l.GetLabels()[to.String()],
		)
	} else {
		f.log.Debug("link",
			"from nodeName", from.String(),
			"from podIndex", from.GetPodIndex(),
			"from ifName", l.GetLabels()[from.String()],
			"to nodeName", to.String(),
			"to podIndex", to.GetPodIndex(),
			"to ifName", l.GetLabels()[to.String()],
		)
	}

}

func (f *fabric) processTier(position topov1alpha1.Position, index uint32, tierTempl *topov1alpha1.TierTemplate, toBeDeployed bool) error {
	vendorNum := len(tierTempl.VendorInfo)
	for n := uint32(0); n < tierTempl.NodeNumber; n++ {
		// venndor Index is used to map to the particular node based on modulo
		// if 1 vendor  -> all nodes are from 1 vendor
		// if 2 vendors -> all odd nodes will be vendor A and all even nodes will be vendor B
		// if 3 vendors -> 1st is vendorA, 2nd vendor B, 3rd is vendor C
		vendorIdx := n % uint32(vendorNum)

		ni := &nodeInfo{
			position:          position,
			graphIndex:        f.graph.NewNode().ID(),
			relativeNodeIndex: n + 1,
			uplinkPerNode:     tierTempl.UplinksPerNode,
			vendorInfo:        tierTempl.VendorInfo[vendorIdx],
			toBeDeployed:      toBeDeployed,
			location:          f.location,
		}

		switch position {
		case topov1alpha1.PositionSuperspine:
			// relativeNodeIndexInPLane: m + 1 -> starts counting from 1, used when multiple nodes are used in the superspine plane
			// PlaneIndex: n + 1 -> could also be called the Plane Index
			ni.planeIndex = index
		case topov1alpha1.PositionBorderLeaf:
			// no plane or podIndex
		default:
			ni.podIndex = index
		}

		n, err := NewNode(ni)
		if err != nil {
			return err
		}
		f.addNode(n)
	}
	return nil
}

func (f *fabric) addNode(n Node) {
	f.graph.AddNode(n)
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

func (f *fabric) addLink(from, to Node) Link {
	l := f.graph.NewLine(from, to)
	return NewLink(l)
}

func (f *fabric) parseTemplate(t *topov1alpha1.FabricTemplate) (*topov1alpha1.FabricTemplate, error) {
	mt := &topov1alpha1.FabricTemplate{}

	if err := t.CheckTemplate(true); err != nil {
		return nil, err
	}

	if t.HasReference() {
		f.log.Debug("parseTemplate", "hasReference", true)
		mt.BorderLeaf = t.BorderLeaf
		mt.Tier1 = t.Tier1
		if t.Settings != nil {
			mt.Settings = &topov1alpha1.FabricTemplateSettings{
				MaxUplinksTier2ToTier1: t.Settings.MaxUplinksTier2ToTier1,
				MaxUplinksTier3ToTier2: t.Settings.MaxUplinksTier3ToTier2,
				MaxSpinesPerPod:        t.Settings.MaxSpinesPerPod,
			}
		}

		mt.Pod = make([]*topov1alpha1.PodTemplate, 0)
		for _, pod := range t.Pod {
			if pod.TemplateRef != nil {
				pd, err := f.getPodDefintionFromTemplate(pod.TemplateRef.Name)
				if err != nil {
					return nil, err
				}
				pd.SetToBeDeployed(true)
				mt.Pod = append(mt.Pod, pd)
			}
			/*
				if pod.DefinitionReference != nil {
					name, namespace := meta.NamespacedName(*pod.DefinitionReference).GetNameAndNamespace()
					t := &topov1alpha1.Definition{}
					if err := f.client.Get(context.TODO(), types.NamespacedName{
						Namespace: namespace,
						Name:      name,
					}, t); err != nil {
						return nil, err
					}
					if len(t.Spec.Properties.Templates) != 1 {
						return nil, fmt.Errorf("definition can only have 1 template")
					}

					pd, err := f.getPodDefintionFromTemplate(t.Spec.Properties.Templates[0].NamespacedName)
					if err != nil {
						return nil, err
					}
					pd.SetToBeDeployed(false)
					mt.Pod = append(mt.Pod, pd)
				}
			*/
		}
	} else {
		mt = t
		for _, pod := range t.Pod {
			pod.SetToBeDeployed(true)
		}
	}

	return mt, nil
}

func (f *fabric) getPodDefintionFromTemplate(name string) (*topov1alpha1.PodTemplate, error) {
	t := &topov1alpha1.Template{}
	if err := f.client.Get(context.TODO(), types.NamespacedName{
		Namespace: f.namespace,
		Name:      name,
	}, t); err != nil {
		return nil, err
	}
	if err := t.Spec.Properties.Fabric.CheckTemplate(false); err != nil {
		return nil, err
	}
	return t.Spec.Properties.Fabric.Pod[0], nil
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

	nodes := f.GetNodes()
	for _, n := range nodes {

		vendorType := ""
		switch n.GetVendorType() {
		case targetv1.VendorTypeNokiaSRL:
			vendorType = "srlinux"
		case targetv1.VendorTypeNokiaSROS:
			vendorType = "sros"
		default:
			vendorType = string(n.GetVendorType())
		}

		t.Nodes = append(t.Nodes, &TopologyJsonNode{
			ID:    int(n.ID()),
			Level: topov1alpha1.GetLevel(topov1alpha1.Position(n.GetPosition())),
			Label: n.String(),
			Nos:   vendorType,
			Cid:   n.GetPosition(),
			Data: &TopologyJsonNodedata{
				Model: n.GetPlatform(),
			},
		})
	}

	links := f.GetLinks()
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
