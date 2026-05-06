package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_YAMLEncode_BlockStyle(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::yamlencode({
						apiVersion = "v1"
						kind = "ConfigMap"
						data = { key = "value" }
					})
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

func TestAcc_YAMLEncode_WithComments(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::yamlencode(
						{ apiVersion = "v1", kind = "ConfigMap" },
						{ comments = { apiVersion = "K8s API version" } }
					)
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

func TestAcc_YAMLEncode_MultilineScript(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::yamlencode({
						data = { script = "#!/bin/bash\necho hello\n" }
					})
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}
