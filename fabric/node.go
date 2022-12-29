package fabric

import (
	"fmt"
	"strconv"
	"strings"

	targetv1 "github.com/henderiw-k8s-lcnc/target/apis/target/v1"
	topov1alpha1 "github.com/henderiw-k8s-lcnc/topology/apis/topo/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
// KeyTier              = "tier"
// KeyPosition          = "position"
// KeyPodIndex          = "podIndex"
// KeyPlaneIndex        = "planeIndex"
// KeyRelativeNodeIndex = "relativeNodeIndex"
)

type Node interface {
	ID() int64
	String() string
	DOTID() string
	GetKRMNode() *topov1alpha1.Node
	//GetPosition() string
	//GetRelativeNodeIndex() string
	//GetPlaneIndex() string
	//GetPodIndex() string
	//GetVendorType() targetv1.VendorType
	//GetPlatform() string
	GetUplinkPerNode() uint32
	GetInterfaceName(idx uint32) string
	GetInterfaceNameWithPlatfromOffset(idx uint32) string
	//IsToBeDeployed() bool
	//GetLocation() *topov1alpha1.Location

	//Attributes() []encoding.Attribute
	//SetLabel(label map[string]string) error
	//UpdateLabel(label map[string]string) error
	GetLabels() labels.Set
}

type originInfo struct {
	name      string
	namespace string
	location  *topov1alpha1.Location
}

type nodeInfo struct {
	//originInfo        *originInfo
	graphIndex        int64
	position          topov1alpha1.Position // tier1, tier2, tier3
	podIndex          uint32                // used for leaf and spines
	planeIndex        uint32                // used for superspines
	relativeNodeIndex uint32                // relative index for the position within the pod (leaf/spine) or plane (superspine)
	uplinkPerNode     uint32
	vendorInfo        *topov1alpha1.FabricTierVendorInfo
	//toBeDeployed      bool
}

func NewNode(oi *originInfo, ni *nodeInfo) Node {
	n := &node{
		graphIndex: ni.graphIndex,
		//relativeNodeIndex: nodeInfo.relativeNodeIndex,
		//vendorInfo:   nodeInfo.vendorInfo,
		//toBeDeployed: nodeInfo.toBeDeployed,
		//location:     nodeInfo.location,
		nodeKRM: buildNode(oi, ni),
	}

	/*
		labels := map[string]string{
			KeyPosition:          string(nodeInfo.position),
			KeyRelativeNodeIndex: strconv.Itoa(int(nodeInfo.relativeNodeIndex)),
		}
	*/

	switch ni.position {
	case topov1alpha1.PositionLeaf, topov1alpha1.PositionSpine:
		//n.podIndex = nodeInfo.podIndex
		n.uplinkPerNode = ni.uplinkPerNode
		//labels[KeyPodIndex] = strconv.Itoa(int(nodeInfo.podIndex))
	case topov1alpha1.PositionSuperspine:
		//n.planeIndex = nodeInfo.planeIndex
		//labels[KeyPlaneIndex] = strconv.Itoa(int(nodeInfo.planeIndex))
	}
	//if err := n.SetLabel(labels); err != nil {
	//	return nil, err
	//}
	return n

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
	//attrs         labels.Set
	//vendorInfo    *topov1alpha1.FabricTierVendorInfo
	uplinkPerNode uint32
	//toBeDeployed  bool
	//location      *topov1alpha1.Location
	nodeKRM *topov1alpha1.Node
}

func (n *node) ID() int64                      { return n.graphIndex }
func (n *node) String() string                 { return n.GetKRMNode().GetName() }
func (n *node) DOTID() string                  { return n.GetKRMNode().GetName() }
func (n *node) GetKRMNode() *topov1alpha1.Node { return n.nodeKRM }

// func (n *node) GetPosition() string                 { return n.GetLabels()[KeyPosition] }
// func (n *node) GetRelativeNodeIndex() string        { return n.GetLabels()[KeyRelativeNodeIndex] }
// func (n *node) GetPlaneIndex() string               { return n.GetLabels()[KeyPlaneIndex] }
// func (n *node) GetPodIndex() string                 { return n.GetLabels()[KeyPodIndex] }
// func (n *node) GetVendorType() targetv1.VendorType  { return n.vendorInfo.VendorType }
// func (n *node) GetPlatform() string                 { return n.vendorInfo.Platform }
func (n *node) GetUplinkPerNode() uint32 { return n.uplinkPerNode }

//func (n *node) IsToBeDeployed() bool                { return n.toBeDeployed }
//func (n *node) GetLocation() *topov1alpha1.Location { return n.location }

func (n *node) GetInterfaceName(idx uint32) string {
	return fmt.Sprintf("int-1/%d", idx)
}

func (n *node) GetInterfaceNameWithPlatfromOffset(idx uint32) string {

	var actualIndex uint32
	switch n.GetKRMNode().Spec.Properties.VendorType {
	case targetv1.VendorTypeNokiaSRL:
		switch n.GetKRMNode().Spec.Properties.Position {
		case topov1alpha1.PositionLeaf:
			switch n.GetKRMNode().Spec.Properties.Platform {
			case "IXR-D3":
				actualIndex = idx + 26
			case "IXR-D2":
				actualIndex = idx + 48
			}
		case topov1alpha1.PositionSpine:
			switch n.GetKRMNode().Spec.Properties.Platform {
			case "IXR-D3":
				actualIndex = idx + 24
			}
		}
	case targetv1.VendorTypeNokiaSROS:
	}

	return fmt.Sprintf("int-1/%d", actualIndex)
}

/*
func (n *node) getName() string {
	switch n.GetPosition() {
	case string(topov1alpha1.PositionSuperspine):
		return fmt.Sprintf("plane%s-%s%s",
			n.GetLabels()[KeyPlaneIndex],
			n.GetLabels()[KeyPosition],
			n.GetLabels()[KeyRelativeNodeIndex],
		)
	case string(topov1alpha1.PositionBorderLeaf):
		return fmt.Sprintf("%s%s",
			n.GetLabels()[KeyPosition],
			n.GetLabels()[KeyRelativeNodeIndex],
		)
	case string(topov1alpha1.PositionSpine), string(topov1alpha1.PositionLeaf):
		return fmt.Sprintf("pod%s-%s%s",
			n.GetLabels()[KeyPodIndex],
			n.GetLabels()[KeyPosition],
			n.GetLabels()[KeyRelativeNodeIndex],
		)
	}
	return "dummy"
}
*/
/*
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
*/

func (n *node) GetLabels() labels.Set { return n.GetKRMNode().GetLabels() }

func buildNode(oi *originInfo, ni *nodeInfo) *topov1alpha1.Node {
	labels := map[string]string{
		topov1alpha1.LabelKeyTopologyPosition:          string(ni.position),
		topov1alpha1.LabelKeyTopologyRelativeNodeIndex: strconv.Itoa(int(ni.relativeNodeIndex)),
		topov1alpha1.LabelKeyTopologyVendorType:        string(ni.vendorInfo.VendorType),
		topov1alpha1.LabelKeyTopologyPlatform:          ni.vendorInfo.Platform,
		//topov1alpha1.LabelKeyOrganization:                           cr.GetOrganization(),
		//LabelKeyDeployment:                             cr.GetDeployment(),
		//LabelKeyAvailabilityZone:                       cr.GetAvailabilityZone(),
		topov1alpha1.LabelKeyTopology: oi.name,
	}
	if ni.position == topov1alpha1.PositionSuperspine {
		labels[topov1alpha1.LabelKeyTopologyPlaneIndex] = strconv.Itoa(int(ni.planeIndex))
	} else {
		labels[topov1alpha1.LabelKeyTopologyPodIndex] = strconv.Itoa(int(ni.podIndex))
	}

	return &topov1alpha1.Node{
		TypeMeta: metav1.TypeMeta{
			APIVersion: topov1alpha1.GroupVersion.String(),
			Kind: topov1alpha1.NodeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.Join([]string{oi.name, ni.getName()}, "."),
			Namespace: oi.namespace,
			Labels:    labels,
			//OwnerReferences: []metav1.OwnerReference{meta.AsController(meta.TypedReferenceTo(cr, topov1alpha1.DefinitionGroupVersionKind))},
		},
		Spec: topov1alpha1.NodeSpec{
			Properties: &topov1alpha1.NodeProperties{
				VendorType: ni.vendorInfo.VendorType,
				Platform:   ni.vendorInfo.Platform,
				Position:   ni.position,
				//MacAddress: ,
				//SerialNumber: ,
				//ExpectedSWVersion: ,
				//MgmtIPAddress: ,
				Location: oi.location,
				// Tags://
			},
		},
	}
}

func (n *nodeInfo) getName() string {
	switch n.position {
	case topov1alpha1.PositionSuperspine:
		return fmt.Sprintf("plane%s-%s%s",
			strconv.Itoa(int(n.planeIndex)),
			string(n.position),
			strconv.Itoa(int(n.relativeNodeIndex)),
		)
	case topov1alpha1.PositionBorderLeaf:
		return fmt.Sprintf("%s%s",
			string(n.position),
			strconv.Itoa(int(n.relativeNodeIndex)),
		)
	case topov1alpha1.PositionSpine, topov1alpha1.PositionLeaf:
		return fmt.Sprintf("pod%s-%s%s",
			strconv.Itoa(int(n.podIndex)),
			string(n.position),
			strconv.Itoa(int(n.relativeNodeIndex)),
		)
	}
	return "dummy"
}
