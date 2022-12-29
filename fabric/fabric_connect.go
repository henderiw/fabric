package fabric

import (
	"fmt"
	"strconv"

	topov1alpha1 "github.com/henderiw-k8s-lcnc/topology/apis/topo/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func (r *fabric) connect(t *topov1alpha1.FabricTemplate) error {

	if err := r.connectSpine2Leaf(t); err != nil {
		return err
	}

	if err := r.connectSpine2SuperSpine(t); err != nil {
		return err
	}

	if err := r.connectSpine2borderLeaf(t); err != nil {
		return err
	}

	return nil
}

func (r *fabric) connectSpine2Leaf(t *topov1alpha1.FabricTemplate) error {
	// process spine-leaf links
	for p, pod := range t.Pod {
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

			tier2Nodes := r.nodesByLabel(tier2Selector)
			tier3Nodes := r.nodesByLabel(tier3Selector)

			for _, tier2Node := range tier2Nodes {
				//tier2NodeIndex := uint32(n) + 1
				for _, tier3Node := range tier3Nodes {
					//tier3NodeIndex := uint32(m) + 1
					// validate if the uplinks per node is not greater than max uplinks
					// otherwise there is a conflict and the algorithm behind will create
					// overlapping indexes
					uplinksPerNode := tier3Node.GetUplinkPerNode()
					if uplinksPerNode > t.Settings.MaxUplinksTier3ToTier2 {
						return fmt.Errorf("uplink per node %d can not be bigger than maxUplinksTier3ToTier2 %d",
							uplinksPerNode, t.Settings.MaxUplinksTier3ToTier2)
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

						l := r.addLink(tier2Node, tier3Node)

						tier3NodeIndex, err := strconv.Atoi(tier3Node.GetRelativeNodeIndex())
						if err != nil {
							return err
						}
						tier2NodeIndex, err := strconv.Atoi(tier2Node.GetRelativeNodeIndex())
						if err != nil {
							return err
						}

						label := map[string]string{
							tier2Node.String(): tier2Node.GetInterfaceName(u + 1 + ((uint32(tier3NodeIndex) - 1) * t.Settings.MaxUplinksTier3ToTier2)),
							tier3Node.String(): tier3Node.GetInterfaceNameWithPlatfromOffset(u + 1 + ((uint32(tier2NodeIndex) - 1) * t.Settings.MaxUplinksTier3ToTier2)),
						}
						l.SetLabel(label)

						r.graph.SetLine(l)

						//f.log.Debug("Adding link", "from:", tier2Node.String(), "itfce", label[tier2Node.String()], "to:", tier3Node.String(), "itfce", label[tier3Node.String()])
					}
				}
			}
		}
	}
	return nil
}

func (r *fabric) connectSpine2SuperSpine(t *topov1alpha1.FabricTemplate) error {
	tier1Selector := labels.NewSelector()
	tier1Req, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionSuperspine)})
	tier1Selector = tier1Selector.Add(*tier1Req)
	tier1Nodes := r.nodesByLabel(tier1Selector)

	tier2Selector := labels.NewSelector()
	tier2Req, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionSpine)})
	tier2Selector = tier2Selector.Add(*tier2Req)
	tier2Nodes := r.nodesByLabel(tier2Selector)

	// process superspine-spine links
	for _, tier1Node := range tier1Nodes {
		for _, tier2Node := range tier2Nodes {
			// validate if the uplinks per node is not greater than max uplinks
			// otherwise there is a conflict and the algorithm behind will create
			// overlapping indexes
			uplinksPerNode := tier2Node.GetUplinkPerNode()
			if uplinksPerNode > t.Settings.MaxUplinksTier2ToTier1 {
				return fmt.Errorf("uplink per node %d can not be bigger than maxUplinksTier2ToTier1 %d", uplinksPerNode, t.Settings.MaxUplinksTier2ToTier1)
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

					l := r.addLink(tier1Node, tier2Node)

					podIndex, err := strconv.Atoi(tier2Node.GetPodIndex())
					if err != nil {
						return err
					}
					relativeIndex, err := strconv.Atoi(tier1Node.GetRelativeNodeIndex())
					if err != nil {
						return err
					}

					label := map[string]string{
						tier1Node.String(): tier1Node.GetInterfaceName(u + 1 + (uint32(podIndex-1) * t.Settings.MaxUplinksTier2ToTier1)),
						tier2Node.String(): tier2Node.GetInterfaceNameWithPlatfromOffset(u + 1 + (uint32(relativeIndex-1) * t.Settings.MaxUplinksTier2ToTier1)),
					}
					l.SetLabel(label)

					r.graph.SetLine(l)

					//f.log.Debug("Adding link", "from:", tier1Node.String(), "itfce", label[tier1Node.String()], "to:", tier2Node.String(), "itfce", label[tier2Node.String()])
				}
			}
		}
	}
	return nil
}

func (r *fabric) connectSpine2borderLeaf(t *topov1alpha1.FabricTemplate) error {
	tier2Selector := labels.NewSelector()
	tier2Req, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionSpine)})
	tier2Selector = tier2Selector.Add(*tier2Req)
	tier2Nodes := r.nodesByLabel(tier2Selector)

	blSelector := labels.NewSelector()
	blReq, _ := labels.NewRequirement(KeyPosition, selection.Equals, []string{string(topov1alpha1.PositionBorderLeaf)})
	blSelector = blSelector.Add(*blReq)
	blNodes := r.nodesByLabel(blSelector)

	// process borderleaf-spine links
	for _, blNode := range blNodes {
		for _, tier2Node := range tier2Nodes {
			// validate if the uplinks per node is not greater than max uplinks
			// otherwise there is a conflict and the algorithm behind will create
			// overlapping indexes
			uplinksPerNode := tier2Node.GetUplinkPerNode()
			if uplinksPerNode > t.Settings.MaxUplinksTier2ToTier1 {
				return fmt.Errorf("uplink per node %d can not be bigger than maxUplinksTier2ToTier1 %d", uplinksPerNode, t.Settings.MaxUplinksTier2ToTier1)
			}

			for u := uint32(0); u < uplinksPerNode; u++ {

				l := r.addLink(blNode, tier2Node)

				podIndex, err := strconv.Atoi(tier2Node.GetPodIndex())
				if err != nil {
					return err
				}
				if uint32(podIndex) > t.Settings.MaxUplinksTier2ToTier1 {
					return fmt.Errorf("spines per pod cannot be bigger than maxSpinesPerPod")
				}
				tier2NodeIndex, err := strconv.Atoi(tier2Node.GetRelativeNodeIndex())
				if err != nil {
					return err
				}
				blNodeIndex, err := strconv.Atoi(blNode.GetRelativeNodeIndex())
				if err != nil {
					return err
				}

				label := map[string]string{
					blNode.String():    blNode.GetInterfaceName(u + 1 + ((uint32(podIndex-1) + ((uint32(tier2NodeIndex) - 1) * t.Settings.MaxSpinesPerPod)) * t.Settings.MaxUplinksTier2ToTier1)),
					tier2Node.String(): tier2Node.GetInterfaceNameWithPlatfromOffset(u + 1 + (uint32(blNodeIndex-1) * t.Settings.MaxUplinksTier2ToTier1)),
				}
				l.SetLabel(label)

				r.graph.SetLine(l)

				//f.log.Debug("Adding link", "from:", blNode.String(), "itfce", label[blNode.String()], "to:", tier2Node.String(), "itfce", label[tier2Node.String()])
			}
		}
	}
	return nil
}
