//go:generate packer-sdc struct-markdown

package common

import (
	"github.com/hashicorp/packer-plugin-sdk/multistep"
)

const (
	SkippingImageCreation = "Skipping image creation..."
)

type Config struct {
	// Skip creating the image.
	// Useful for setting to `true` during a build test stage.
	// Defaults to `false`.
	SkipCreateImage bool `mapstructure:"skip_create_image" required:"false"`
}

// CaptureSteps returns the steps unless `SkipCreateImage` is `true`. In that case it returns
// a step that to inform the user that image capture is being skipped.
func (config Config) CaptureSteps(say func(string), steps ...multistep.Step) []multistep.Step {
	if !config.SkipCreateImage {
		return steps
	}

	return []multistep.Step{
		&StepNotify{
			message: SkippingImageCreation,
			say:     say,
		},
	}
}
