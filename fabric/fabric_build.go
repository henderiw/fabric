package fabric

import (
	"fmt"

	topov1alpha1 "github.com/henderiw-k8s-lcnc/topology/apis/topo/v1alpha1"
)

// buildNewFabricTemplate builds a new fabric template based on the master and child template
// references
func (r *fabric) buildNewFabricTemplate() (*topov1alpha1.FabricTemplate, error) {
	if len(r.cfg.MasterTemplates) > 0 {
		ft := r.cfg.MasterTemplates[0].Spec.Properties.Fabric
		//f.log.Debug("parseTemplate", "hasReference", true)
		newt := ft.DeepCopy()

		newt.Pod = make([]*topov1alpha1.PodTemplate, 0)
		for _, pod := range ft.Pod {
			if pod.TemplateRef != nil {
				pd, err := r.getPodFromTemplate(pod.TemplateRef)
				if err != nil {
					return nil, err
				}
				newt.Pod = append(newt.Pod, pd)
			}
		}
		return newt, nil
	}
	return r.cfg.ChildTemplates[0].Spec.Properties.Fabric.DeepCopy(), nil
}

func (r *fabric) getPodFromTemplate(podTemplateRef *topov1alpha1.TemplateReference) (*topov1alpha1.PodTemplate, error) {
	for _, t := range r.cfg.ChildTemplates {
		if t.GetName() == podTemplateRef.Name && t.GetNamespace() == podTemplateRef.Namespace {
			// we validates the child templates to ensure they have only 1 podTemplate
			// so we are ok to use index 0
			return t.Spec.Properties.Fabric.Pod[0], nil
		}
	}
	return nil, fmt.Errorf("no podTemplate reference found for %v in %v", podTemplateRef, r.cfg.ChildTemplates)
}
