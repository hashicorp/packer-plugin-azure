package common

import "os"

const AzureDebugLogsEnvVar string = "PACKER_AZURE_DEBUG_LOG"

func IsDebugEnabled() bool {
	debug, defined := os.LookupEnv(AzureDebugLogsEnvVar)
	if !defined {
		return false
	}

	return debug != ""
}
