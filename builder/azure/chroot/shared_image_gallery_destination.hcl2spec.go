// Code generated by "packer-sdc mapstructure-to-hcl2"; DO NOT EDIT.

package chroot

import (
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"
)

// FlatSharedImageGalleryDestination is an auto-generated flat version of SharedImageGalleryDestination.
// Where the contents of a field with a `mapstructure:,squash` tag are bubbled up.
type FlatSharedImageGalleryDestination struct {
	ResourceGroup         *string            `mapstructure:"resource_group" required:"true" cty:"resource_group" hcl:"resource_group"`
	GalleryName           *string            `mapstructure:"gallery_name" required:"true" cty:"gallery_name" hcl:"gallery_name"`
	ImageName             *string            `mapstructure:"image_name" required:"true" cty:"image_name" hcl:"image_name"`
	ImageVersion          *string            `mapstructure:"image_version" required:"true" cty:"image_version" hcl:"image_version"`
	TargetRegions         []FlatTargetRegion `mapstructure:"target_regions" cty:"target_regions" hcl:"target_regions"`
	ExcludeFromLatest     *bool              `mapstructure:"exclude_from_latest" cty:"exclude_from_latest" hcl:"exclude_from_latest"`
	ExcludeFromLatestTypo *bool              `mapstructure:"exlude_from_latest" undocumented:"true" cty:"exlude_from_latest" hcl:"exlude_from_latest"`
}

// FlatMapstructure returns a new FlatSharedImageGalleryDestination.
// FlatSharedImageGalleryDestination is an auto-generated flat version of SharedImageGalleryDestination.
// Where the contents a fields with a `mapstructure:,squash` tag are bubbled up.
func (*SharedImageGalleryDestination) FlatMapstructure() interface{ HCL2Spec() map[string]hcldec.Spec } {
	return new(FlatSharedImageGalleryDestination)
}

// HCL2Spec returns the hcl spec of a SharedImageGalleryDestination.
// This spec is used by HCL to read the fields of SharedImageGalleryDestination.
// The decoded values from this spec will then be applied to a FlatSharedImageGalleryDestination.
func (*FlatSharedImageGalleryDestination) HCL2Spec() map[string]hcldec.Spec {
	s := map[string]hcldec.Spec{
		"resource_group":      &hcldec.AttrSpec{Name: "resource_group", Type: cty.String, Required: false},
		"gallery_name":        &hcldec.AttrSpec{Name: "gallery_name", Type: cty.String, Required: false},
		"image_name":          &hcldec.AttrSpec{Name: "image_name", Type: cty.String, Required: false},
		"image_version":       &hcldec.AttrSpec{Name: "image_version", Type: cty.String, Required: false},
		"target_regions":      &hcldec.BlockListSpec{TypeName: "target_regions", Nested: hcldec.ObjectSpec((*FlatTargetRegion)(nil).HCL2Spec())},
		"exclude_from_latest": &hcldec.AttrSpec{Name: "exclude_from_latest", Type: cty.Bool, Required: false},
		"exlude_from_latest":  &hcldec.AttrSpec{Name: "exlude_from_latest", Type: cty.Bool, Required: false},
	}
	return s
}

// FlatTargetRegion is an auto-generated flat version of TargetRegion.
// Where the contents of a field with a `mapstructure:,squash` tag are bubbled up.
type FlatTargetRegion struct {
	Name               *string `mapstructure:"name" required:"true" cty:"name" hcl:"name"`
	ReplicaCount       *int64  `mapstructure:"replicas" cty:"replicas" hcl:"replicas"`
	StorageAccountType *string `mapstructure:"storage_account_type" cty:"storage_account_type" hcl:"storage_account_type"`
}

// FlatMapstructure returns a new FlatTargetRegion.
// FlatTargetRegion is an auto-generated flat version of TargetRegion.
// Where the contents a fields with a `mapstructure:,squash` tag are bubbled up.
func (*TargetRegion) FlatMapstructure() interface{ HCL2Spec() map[string]hcldec.Spec } {
	return new(FlatTargetRegion)
}

// HCL2Spec returns the hcl spec of a TargetRegion.
// This spec is used by HCL to read the fields of TargetRegion.
// The decoded values from this spec will then be applied to a FlatTargetRegion.
func (*FlatTargetRegion) HCL2Spec() map[string]hcldec.Spec {
	s := map[string]hcldec.Spec{
		"name":                 &hcldec.AttrSpec{Name: "name", Type: cty.String, Required: false},
		"replicas":             &hcldec.AttrSpec{Name: "replicas", Type: cty.Number, Required: false},
		"storage_account_type": &hcldec.AttrSpec{Name: "storage_account_type", Type: cty.String, Required: false},
	}
	return s
}
