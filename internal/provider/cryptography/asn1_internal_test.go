package cryptography

import (
	"bytes"
	"strings"
	"testing"
)

// TestDecodeASN1RejectsExcessiveNodeCount drives the node-count cap directly from Go because expressing >100k base64-encoded TLVs through HCL acceptance tests is impractical (HCL string literals get unwieldy and Terraform's plan-time format helpers don't easily produce raw bytes containing 0x05 0x00 NULL TLVs). A flat SEQUENCE { N × NULL } slips past the depth cap, so without the node-count cap an adversarial blob with millions of NULL children would allocate millions of Terraform ObjectValue / DynamicValue wrappers and OOM the provider.
func TestDecodeASN1RejectsExcessiveNodeCount(t *testing.T) {
	// asn1MaxNodes = 100_000. Build SEQUENCE { 100_001 × NULL } so the count exceeds the cap (root SEQUENCE = 1 node + 100_001 children = 100_002 total).
	const childCount = 100_001
	body := bytes.Repeat([]byte{0x05, 0x00}, childCount)
	// SEQUENCE tag 0x30, length encoded long-form: body is 200_002 bytes (0x30D02), so length is 3 bytes (0x83 0x03 0x0D 0x02).
	bodyLen := len(body)
	der := append([]byte{0x30, 0x83, byte(bodyLen >> 16), byte(bodyLen >> 8), byte(bodyLen)}, body...)

	_, err := decodeASN1(der)
	if err == nil {
		t.Fatal("expected node-count cap error; got nil")
	}
	if !strings.Contains(err.Error(), "more than") || !strings.Contains(err.Error(), "nodes") {
		t.Fatalf("expected node-count error message, got: %v", err)
	}
}

// TestDecodeASN1AcceptsNodeCountUnderCap is the positive companion: a flat SEQUENCE just under the cap should decode without error.
func TestDecodeASN1AcceptsNodeCountUnderCap(t *testing.T) {
	const childCount = 50_000
	body := bytes.Repeat([]byte{0x05, 0x00}, childCount)
	bodyLen := len(body)
	der := append([]byte{0x30, 0x83, byte(bodyLen >> 16), byte(bodyLen >> 8), byte(bodyLen)}, body...)

	if _, err := decodeASN1(der); err != nil {
		t.Fatalf("decode under cap unexpectedly errored: %v", err)
	}
}
