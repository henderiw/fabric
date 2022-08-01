package fabric

import (
	"fmt"
	"sort"
	"strconv"

	targetv1 "github.com/yndd/target/apis/target/v1"
	topov1alpha1 "github.com/yndd/topology/apis/topo/v1alpha1"
	"gonum.org/v1/gonum/graph/encoding"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	KeyTier              = "tier"
	KeyPosition          = "position"
	KeyPodIndex          = "podIndex"
	KeyPlaneIndex        = "planeIndex"
	KeyRelativeNodeIndex = "relativeNodeIndex"
)

type Node interface {
	ID() int64
	String() string
	DOTID() string
	GetPosition() string
	GetRelativeNodeIndex() string
	GetPlaneIndex() string
	GetPodIndex() string
	GetVendorType() targetv1.VendorType
	GetPlatform() string
	GetUplinkPerNode() uint32
	GetInterfaceName(idx uint32) string
	GetInterfaceNameWithPlatfromOffset(idx uint32) string
	IsToBeDeployed() bool

	Attributes() []encoding.Attribute
	SetLabel(label map[string]string) error
	UpdateLabel(label map[string]string) error
	GetLabels() labels.Set
}

type nodeInfo struct {
	graphIndex        int64
	tier              string // tier1, tier2, tier3
	podIndex          uint32 // used for leaf and spines
	planeIndex        uint32 // used for superspines
	relativeNodeIndex uint32 // relative index for the position within the pod (leaf/spine) or plane (superspine)
	uplinkPerNode     uint32
	vendorInfo        *topov1alpha1.FabricTierVendorInfo
	toBeDeployed      bool
}

func NewNode(nodeInfo *nodeInfo) (Node, error) {

	var position topov1alpha1.Position
	switch nodeInfo.tier {
	case "tier1":
		position = topov1alpha1.PositionSuperspine
	case "tier2":
		position = topov1alpha1.PositionSpine
	case "tier3":
		position = topov1alpha1.PositionLeaf
	}

	n := &node{
		graphIndex: nodeInfo.graphIndex,
		//relativeNodeIndex: nodeInfo.relativeNodeIndex,
		vendorInfo:   nodeInfo.vendorInfo,
		toBeDeployed: nodeInfo.toBeDeployed,
	}

	labels := map[string]string{
		KeyPosition:          string(position),
		KeyRelativeNodeIndex: strconv.Itoa(int(nodeInfo.relativeNodeIndex)),
	}

	switch position {
	case topov1alpha1.PositionLeaf, topov1alpha1.PositionSpine:
		//n.podIndex = nodeInfo.podIndex
		n.uplinkPerNode = nodeInfo.uplinkPerNode
		labels[KeyPodIndex] = strconv.Itoa(int(nodeInfo.podIndex))
	case topov1alpha1.PositionSuperspine:
		//n.planeIndex = nodeInfo.planeIndex
		labels[KeyPlaneIndex] = strconv.Itoa(int(nodeInfo.planeIndex))
	}
	if err := n.SetLabel(labels); err != nil {
		return nil, err
	}
	return n, nil

}

type node struct {
	//log      logging.Logger
	//position   topov1alpha1.Position
	graphIndex int64
	// for superspines this is the plane Index
	// for spines/leafs this is the node index within the pod
	//relativeNodeIndex uint32 // relative number within the position/pod
	// only used for leafs and spines
	//podIndex uint32
	// only for superspines
	//planeIndex    uint32
	attrs         labels.Set
	vendorInfo    *topov1alpha1.FabricTierVendorInfo
	uplinkPerNode uint32
	toBeDeployed  bool
}

func (n *node) ID() int64                          { return n.graphIndex }
func (n *node) String() string                     { return n.getName() }
func (n *node) DOTID() string                      { return n.getName() }
func (n *node) GetPosition() string                { return n.GetLabels()[KeyPosition] }
func (n *node) GetRelativeNodeIndex() string       { return n.GetLabels()[KeyRelativeNodeIndex] }
func (n *node) GetPlaneIndex() string              { return n.GetLabels()[KeyPlaneIndex] }
func (n *node) GetPodIndex() string                { return n.GetLabels()[KeyPodIndex] }
func (n *node) GetVendorType() targetv1.VendorType { return n.vendorInfo.VendorType }
func (n *node) GetPlatform() string                { return n.vendorInfo.Platform }
func (n *node) GetUplinkPerNode() uint32           { return n.uplinkPerNode }
func (n *node) IsToBeDeployed() bool               { return n.toBeDeployed }

func (n *node) GetInterfaceName(idx uint32) string {
	return fmt.Sprintf("int-1/%d", idx)
}

func (n *node) GetInterfaceNameWithPlatfromOffset(idx uint32) string {

	var actualIndex uint32
	switch n.GetVendorType() {
	case targetv1.VendorTypeNokiaSRL:
		switch n.GetPosition() {
		case string(topov1alpha1.PositionLeaf):
			switch n.GetPlatform() {
			case "IXR-D3":
				actualIndex = idx + 26
			case "IXR-D2":
				actualIndex = idx + 48
			}
		case string(topov1alpha1.PositionSpine):
			switch n.GetPlatform() {
			case "IXR-D3":
				actualIndex = idx + 24
			}
		}
	case targetv1.VendorTypeNokiaSROS:
	}

	return fmt.Sprintf("int-1/%d", actualIndex)
}

func (n *node) getName() string {
	if n.GetPosition() == string(topov1alpha1.PositionSuperspine) {
		return fmt.Sprintf("plane%s-%s%s",
			n.GetLabels()[KeyPlaneIndex],
			n.GetLabels()[KeyPosition],
			n.GetLabels()[KeyRelativeNodeIndex],
		)
	} else {
		return fmt.Sprintf("pod%s-%s%s",
			n.GetLabels()[KeyPodIndex],
			n.GetLabels()[KeyPosition],
			n.GetLabels()[KeyRelativeNodeIndex],
		)
	}
}

// Attributes implements the encoding.Attributer interface.
func (n *node) Attributes() []encoding.Attribute {
	var keys []string
	for key := range n.attrs {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var attrs []encoding.Attribute
	for _, key := range keys {
		attr := encoding.Attribute{Key: key, Value: n.attrs[key]}
		attrs = append(attrs, attr)
	}
	return attrs
}
func (n *node) SetLabel(label map[string]string) error {
	n.attrs = labels.Set(label)
	return nil
}
func (n *node) UpdateLabel(label map[string]string) error {
	n.attrs = labels.Merge(labels.Set(label), n.attrs)
	return nil
}

func (n *node) GetLabels() labels.Set { return n.attrs }
