package dataformat

import (
	"context"
	"encoding/base64"
	"math/big"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runCBORDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &CBORDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(input)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}
	return resp.Result.Value().(types.Dynamic), nil
}

func TestCBORDecode_ByteStringBecomesBase64String(t *testing.T) {
	bin := []byte{0xca, 0xfe, 0xba, 0xbe}
	em, _ := cbor.CoreDetEncOptions().EncMode()
	raw, err := em.Marshal(map[string]interface{}{"data": bin})
	if err != nil {
		t.Fatalf("setup encode: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	got, ferr := runCBORDecode(t, encoded)
	if ferr != nil {
		t.Fatalf("decode error: %v", ferr)
	}
	obj := got.UnderlyingValue().(types.Object)
	v, ok := obj.Attributes()["data"].(types.String)
	if !ok {
		t.Fatalf("expected data to be String (base64), got %T — plist tag bug regressed?", obj.Attributes()["data"])
	}
	want := base64.StdEncoding.EncodeToString(bin)
	if v.ValueString() != want {
		t.Errorf("want %q, got %q", want, v.ValueString())
	}
}

func TestCBORDecode_DatetimeBecomesRFC3339String(t *testing.T) {
	// Encode with tag-0 RFC 3339 strings — the default encoder writes time.Time as a bare Unix-epoch float (no tag), which is meaningfully different from a tagged datetime. We're testing decode of tagged datetimes specifically, so the test setup needs to actually emit one.
	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	em, _ := cbor.EncOptions{
		Time:    cbor.TimeRFC3339,
		TimeTag: cbor.EncTagRequired,
	}.EncMode()
	raw, err := em.Marshal(map[string]interface{}{"when": ts})
	if err != nil {
		t.Fatalf("setup encode: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	got, ferr := runCBORDecode(t, encoded)
	if ferr != nil {
		t.Fatalf("decode error: %v", ferr)
	}
	obj := got.UnderlyingValue().(types.Object)
	v, ok := obj.Attributes()["when"].(types.String)
	if !ok {
		t.Fatalf("expected when to be String (RFC 3339), got %T — plist tag bug regressed?", obj.Attributes()["when"])
	}
	want := ts.Format(time.RFC3339)
	if v.ValueString() != want {
		t.Errorf("want %q, got %q", want, v.ValueString())
	}
}

func TestCBORDecode_BignumBecomesNumber(t *testing.T) {
	// Tag-2 (positive bignum): 2^65 = 36893488147419103232 — too big for uint64. Without explicit handling, fxamacker/cbor decodes this to big.Int which goToTerraformValue can't process. We unwrap to json.Number for full-precision Terraform numbers.
	em, _ := cbor.EncOptions{}.EncMode()
	huge := new(big.Int).Lsh(big.NewInt(1), 65)
	raw, err := em.Marshal(huge)
	if err != nil {
		t.Fatalf("setup encode: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(raw)

	got, ferr := runCBORDecode(t, encoded)
	if ferr != nil {
		t.Fatalf("decode error: %v", ferr)
	}
	n, ok := got.UnderlyingValue().(types.Number)
	if !ok {
		t.Fatalf("expected Number for bignum, got %T", got.UnderlyingValue())
	}
	if n.ValueBigFloat().Cmp(new(big.Float).SetInt(huge)) != 0 {
		t.Errorf("precision loss: want %s, got %s", huge.String(), n.ValueBigFloat().Text('f', -1))
	}
}

func TestCBORDecode_InvalidBase64(t *testing.T) {
	_, err := runCBORDecode(t, "not base64!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}
