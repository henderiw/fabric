package template

/*
import (
	targetv1 "github.com/yndd/target/apis/target/v1"
)

type FabricTemplate struct {
	// superspine
	Tier1      *TierTemplate  `json:"tier1,omitempty"`
	BorderLeaf *TierTemplate  `json:"borderLeaf,omitempty"`
	Pod        []*PodTemplate `json:"pod,omitempty"`
	// max number of uplink per node to the next tier
	// default should be 1 and max is 4
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4
	// +kubebuilder:default=1
	MaxUplinksTier2ToTier1 uint32 `json:"maxUplinksTier2ToTier1,omitempty"`
	// max number of uplink per node to the next tier
	// default should be 1 and max is 4
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4
	// +kubebuilder:default=1
	MaxUplinksTier3ToTier2 uint32            `json:"maxUplinksTier3ToTier2,omitempty"`
	Tag                    map[string]string `json:"tag,omitempty"`
}

type PodTemplate struct {
	// number of pods defined based on this template
	// no default since templates should not define the pod number
	// default should be 1 and max is 16
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=16
	PodNumber *uint32 `json:"num,omitempty"`
	// Tier2 template, that defines the spine parameters in the pod definition
	Tier2 *TierTemplate `json:"tier2,omitempty"`
	// Tier3 template, that defines the leaf parameters in the pod definition
	Tier3 *TierTemplate `json:"tier3,omitempty"`
	// template reference to a template that defines the pod definition
	TemplateReference *string `json:"templateRef,omitempty"`
	// definition reference to a template that defines the pod definition
	DefinitionReference *string           `json:"definitionRef,omitempty"`
	Tag                 map[string]string `json:"tag,omitempty"`
}

type TierTemplate struct {
	// list to support multiple vendors in a tier - typically criss-cross
	VendorInfo []*FabricTierVendorInfo `json:"vendorInfo,omitempty"`
	// number of nodes in the tier
	// for superspine it is the number of spines in a spine plane
	NodeNumber uint32 `json:"num,omitempty"`
	// number of uplink per node to the next tier
	// default should be 1 and max is 4
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=4
	UplinksPerNode uint32            `json:"uplinkPerNode,omitempty"`
	Tag            map[string]string `json:"tag,omitempty"`
}

type FabricTierVendorInfo struct {
	Platform   string              `json:"platform,omitempty"`
	VendorType targetv1.VendorType `json:"vendorType,omitempty"`
	Tag        map[string]string   `json:"tag,omitempty"`
}

*/
