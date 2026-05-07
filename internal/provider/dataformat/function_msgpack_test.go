package dataformat

import (
	"bytes"
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/vmihailenco/msgpack/v5"
)

func runMsgpackDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &MsgpackDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(input)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}
	return resp.Result.Value().(types.Dynamic), nil
}

func TestMsgpackDecode_BinaryFieldBecomesBase64String(t *testing.T) {
	// A msgpack `bin` field must come out as a plain base64 string, NOT a plist-tagged object. This was a real bug pre-review.
	bin := []byte{0xde, 0xad, 0xbe, 0xef}
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)
	enc.SetSortMapKeys(true)
	if err := enc.Encode(map[string]interface{}{"data": bin}); err != nil {
		t.Fatalf("setup encode: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	got, ferr := runMsgpackDecode(t, encoded)
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

func TestMsgpackDecode_TimestampBecomesRFC3339String(t *testing.T) {
	// Same plist-tag concern for the msgpack timestamp extension.
	ts := time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	enc.UseCompactInts(true)
	enc.SetSortMapKeys(true)
	if err := enc.Encode(map[string]interface{}{"when": ts}); err != nil {
		t.Fatalf("setup encode: %v", err)
	}
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())

	got, ferr := runMsgpackDecode(t, encoded)
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

func TestMsgpackDecode_InvalidBase64(t *testing.T) {
	_, err := runMsgpackDecode(t, "not base64!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}
