package arm

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-azure/builder/azure/common"
	"github.com/hashicorp/packer-plugin-azure/builder/azure/common/constants"
	"github.com/hashicorp/packer-plugin-sdk/multistep"
	packersdk "github.com/hashicorp/packer-plugin-sdk/packer"
)

type StepCertificateInKeyVault struct {
	config      *Config
	client      common.AZVaultClientIface
	say         func(message string)
	error       func(e error)
	certificate string
}

func NewStepCertificateInKeyVault(cli common.AZVaultClientIface, ui packersdk.Ui, config *Config, certificate string) *StepCertificateInKeyVault {
	var step = &StepCertificateInKeyVault{
		client:      cli,
		config:      config,
		say:         func(message string) { ui.Say(message) },
		error:       func(e error) { ui.Error(e.Error()) },
		certificate: certificate,
	}

	return step
}

func (s *StepCertificateInKeyVault) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	s.say("Setting the certificate in the KeyVault...")
	var keyVaultName = state.Get(constants.ArmKeyVaultName).(string)

	err := s.client.SetSecret(keyVaultName, DefaultSecretName, s.certificate)
	if err != nil {
		s.error(fmt.Errorf("Error setting winrm cert in custom keyvault: %s", err))
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (*StepCertificateInKeyVault) Cleanup(multistep.StateBag) {
}
