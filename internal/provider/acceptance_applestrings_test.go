package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_AppleStringsDecode_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::applestringsdecode("\"hello\" = \"Hello\";\n\"bye\" = \"Goodbye\";\n")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"hello": knownvalue.StringExact("Hello"),
			"bye":   knownvalue.StringExact("Goodbye"),
		})),
	)
}

func TestAcc_AppleStringsDecode_WithComments(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::applestringsdecode("/* greeting */\n\"hello\" = \"Hi\";\n// trailing\n\"bye\" = \"Bye\";\n")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"hello": knownvalue.StringExact("Hi"),
			"bye":   knownvalue.StringExact("Bye"),
		})),
	)
}

func TestAcc_AppleStringsDecode_EscapeSequences(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::applestringsdecode("\"k\" = \"line1\\nline2\\u0021\";\n")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"k": knownvalue.StringExact("line1\nline2!"),
		})),
	)
}

func TestAcc_AppleStringsEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::applestringsencode({ greeting = "Hello", farewell = "Bye" })
		}`,
		// Sorted keys: farewell before greeting.
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("\"farewell\" = \"Bye\";\n\"greeting\" = \"Hello\";\n")),
	)
}

func TestAcc_AppleStringsDecode_Malformed(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::applestringsdecode("\"key\" = \"unterminated") }`,
		regexp.MustCompile(`(?i)strings|unterminated|expected`),
	)
}
