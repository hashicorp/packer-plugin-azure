package common_test

import (
	"context"
	"testing"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/stretchr/testify/assert"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
)

func TestStepNotify(t *testing.T) {
	var said []string

	say := func(what string) {
		said = append(said, what)
	}

	message := "Notify Step"

	step := common.NewStepNotify(message, say)
	state := &multistep.BasicStateBag{}

	ctx := context.Background()

	step.Run(ctx, state)

	assert.Equal(t, said, []string{message})
}
