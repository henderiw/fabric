package fabric

import topov1alpha1 "github.com/henderiw-k8s-lcnc/topology/apis/topo/v1alpha1"

func (r *fabric) populateNodes(t *topov1alpha1.FabricTemplate) error {
	// process leaf/spine nodes
	// p is number of pod templates
	for p, pod := range t.Pod {
		// i is the number of pods in a template
		for i := uint32(0); i < pod.GetPodNumber(); i++ {
			// podIndex is pod template index * pod index within the template
			podIndex := (uint32(p) + 1) * (i + 1)

			// tier 2 -> spines in the pod
			if err := r.processTier(topov1alpha1.PositionSpine, podIndex, pod.Tier2); err != nil {
				return err
			}
			// tier 3 -> leafs in the pod
			if err := r.processTier(topov1alpha1.PositionLeaf, podIndex, pod.Tier3); err != nil {
				return err
			}
		}
	}

	// proces superspines
	// the superspine is equal to the amount of spines per pod and multiplied with the number in the template
	if t.Tier1 != nil {
		// process superspine nodes
		for n := uint32(0); n < t.GetSuperSpines(); n++ {
			if err := r.processTier(topov1alpha1.PositionSuperspine, n+1, t.Tier1); err != nil {
				return err
			}
		}
	}

	// process borderleafs
	if t.BorderLeaf != nil {
		// process borderleafs nodes
		if err := r.processTier(topov1alpha1.PositionBorderLeaf, 1, t.BorderLeaf); err != nil {
			return err
		}
	}
	return nil
}

func (r *fabric) processTier(position topov1alpha1.Position, index uint32, tierTempl *topov1alpha1.TierTemplate) error {
	vendorNum := len(tierTempl.VendorInfo)
	for n := uint32(0); n < tierTempl.NodeNumber; n++ {
		// venndor Index is used to map to the particular node based on modulo
		// if 1 vendor  -> all nodes are from 1 vendor
		// if 2 vendors -> all odd nodes will be vendor A and all even nodes will be vendor B
		// if 3 vendors -> 1st is vendorA, 2nd vendor B, 3rd is vendor C
		vendorIdx := n % uint32(vendorNum)

		ni := &nodeInfo{
			position:          position,
			graphIndex:        r.graph.NewNode().ID(),
			relativeNodeIndex: n + 1,
			uplinkPerNode:     tierTempl.UplinksPerNode,
			vendorInfo:        tierTempl.VendorInfo[vendorIdx],
			//toBeDeployed:      toBeDeployed,
			location:          r.cfg.Location,
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
		r.addNode(n)
	}
	return nil
}


