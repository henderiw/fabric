package fabric

import "fmt"

// validateTemplates validates the presence of the templates and its validity
func (r *fabric) validateTemplates() error {
	if len(r.cfg.MasterTemplates) == 0 && len(r.cfg.ChildTemplates) == 0 {
		return fmt.Errorf("cannot build a fabric w/o a template")
	}
	if len(r.cfg.MasterTemplates) > 1 {
		return fmt.Errorf("cannot have multiple master templates: got %v", r.cfg.MasterTemplates)
	}
	if len(r.cfg.MasterTemplates) == 0 && len(r.cfg.ChildTemplates) != 1 {
		return fmt.Errorf("cannot have multiple child templates, when master template is not present: got %v", r.cfg.ChildTemplates)
	}

	for _, t := range r.cfg.MasterTemplates {
		if err := t.Spec.Properties.Fabric.CheckTemplate(true); err != nil {
			return fmt.Errorf("validation pf parent template %s failed, error: %s", t.GetNamespace()+"/"+t.GetName(), err.Error())
		}
	}
	for _, t := range r.cfg.ChildTemplates {
		if err := t.Spec.Properties.Fabric.CheckTemplate(false); err != nil {
			return fmt.Errorf("validation pf parent template %s failed, error: %s", t.GetNamespace()+"/"+t.GetName(), err.Error())
		}
	}
	return nil
}
