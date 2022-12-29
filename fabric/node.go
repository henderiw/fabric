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

type Node interface {
	ID() int64
	String() string
	DOTID() string
	GetKRMNode() *topov1alpha1.Node

	GetUplinkPerNode() uint32
	GetInterfaceName(idx uint32) string
	GetInterfaceNameWithPlatfromOffset(idx uint32) string

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
		nodeKRM:    ni.buildNode(oi),
	}

	if ni.position == topov1alpha1.PositionLeaf || ni.position == topov1alpha1.PositionSpine {
		n.uplinkPerNode = ni.uplinkPerNode
	}

	return n
}

type node struct {
	graphIndex    int64
	uplinkPerNode uint32
	nodeKRM       *topov1alpha1.Node
}

func (n *node) ID() int64 { return n.graphIndex }
func (n *node) String() string {
	switch n.GetKRMNode().GetPosition() {
	case topov1alpha1.PositionSuperspine:
		return fmt.Sprintf("plane%s-%s%s",
			n.GetKRMNode().GetPlaneIndex(),
			n.GetKRMNode().GetPosition(),
			n.GetKRMNode().GetRelativeNodeIndex())
	case topov1alpha1.PositionBorderLeaf:
		return fmt.Sprintf("%s%s",
			n.GetKRMNode().GetPosition(),
			n.GetKRMNode().GetRelativeNodeIndex())
	case topov1alpha1.PositionSpine, topov1alpha1.PositionLeaf:
		return fmt.Sprintf("pod%s-%s%s",
			n.GetKRMNode().GetPodIndex(),
			n.GetKRMNode().GetPosition(),
			n.GetKRMNode().GetRelativeNodeIndex())
	}
	return "dummy"

}
func (n *node) DOTID() string                  { return n.GetKRMNode().GetName() }
func (n *node) GetKRMNode() *topov1alpha1.Node { return n.nodeKRM }
func (n *node) GetLabels() labels.Set          { return n.GetKRMNode().GetLabels() }

func (n *node) GetUplinkPerNode() uint32 { return n.uplinkPerNode }
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

func (ni *nodeInfo) buildNode(oi *originInfo) *topov1alpha1.Node {
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
			Kind:       topov1alpha1.NodeKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.Join([]string{oi.name, ni.getName()}, "."),
			Namespace: oi.namespace,
			Labels:    labels,
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

func (ni *nodeInfo) getName() string {
	switch ni.position {
	case topov1alpha1.PositionSuperspine:
		return fmt.Sprintf("plane%s-%s%s",
			strconv.Itoa(int(ni.planeIndex)),
			string(ni.position),
			strconv.Itoa(int(ni.relativeNodeIndex)),
		)
	case topov1alpha1.PositionBorderLeaf:
		return fmt.Sprintf("%s%s",
			string(ni.position),
			strconv.Itoa(int(ni.relativeNodeIndex)),
		)
	case topov1alpha1.PositionSpine, topov1alpha1.PositionLeaf:
		return fmt.Sprintf("pod%s-%s%s",
			strconv.Itoa(int(ni.podIndex)),
			string(ni.position),
			strconv.Itoa(int(ni.relativeNodeIndex)),
		)
	}
	return "dummy"
}
