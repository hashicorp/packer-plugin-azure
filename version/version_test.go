package version

import (
	"fmt"
	"testing"
)

func TestAzurePluginVersion_FormattedVersion(t *testing.T) {
	if AzurePluginVersion == nil {
		t.Fatal("Unable to continue with nil version")
	}

	expected := Version
	if VersionPrerelease != "" {
		expected = fmt.Sprintf("%s-%s", Version, VersionPrerelease)
	}
	got := AzurePluginVersion.FormattedVersion()
	if got != expected {
		t.Errorf("calling FormattedVersion on AzurePluginVersion failed: expected %s, but got %s", expected, got)
	}

}
