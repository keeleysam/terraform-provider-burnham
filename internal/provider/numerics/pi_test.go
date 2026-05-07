package numerics

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runPiDigit(t *testing.T, n int64) (string, *function.FuncError) {
	t.Helper()
	f := &PiDigitFunction{}
	args := function.NewArgumentsData([]attr.Value{types.Int64Value(n)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	result, ok := resp.Result.Value().(types.String)
	if !ok {
		t.Fatalf("expected String result, got %T", resp.Result.Value())
	}
	return result.ValueString(), nil
}

func runPiDigits(t *testing.T, count int64) (string, *function.FuncError) {
	t.Helper()
	f := &PiDigitsFunction{}
	args := function.NewArgumentsData([]attr.Value{types.Int64Value(count)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringValue(""))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	result, ok := resp.Result.Value().(types.String)
	if !ok {
		t.Fatalf("expected String result, got %T", resp.Result.Value())
	}
	return result.ValueString(), nil
}

func TestPiDigit_BasicReplies(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{1, "1:1"},
		{2, "2:4"},
		{3, "3:1"},
		{10, "10:5"},
		{100, "100:9"},
		{1000, "1000:9"},
	}
	for _, c := range cases {
		got, err := runPiDigit(t, c.n)
		if err != nil {
			t.Errorf("pi_digit(%d) errored: %s", c.n, err.Text)
			continue
		}
		if got != c.want {
			t.Errorf("pi_digit(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestPiDigit_RFCReplyShape(t *testing.T) {
	// Per RFC 3091 §2.1.2 ABNF: reply = nth_digit ":" DIGIT.
	// Verify exactly one ':', the requested n is echoed exactly, and the
	// trailing single character is a digit. No whitespace anywhere.
	for _, n := range []int64{1, 7, 42, 999, 12345, 999_999} {
		got, err := runPiDigit(t, n)
		if err != nil {
			t.Errorf("pi_digit(%d) errored: %s", n, err.Text)
			continue
		}
		if strings.ContainsAny(got, " \t\r\n") {
			t.Errorf("pi_digit(%d) = %q contains whitespace", n, got)
		}
		if strings.Count(got, ":") != 1 {
			t.Errorf("pi_digit(%d) = %q must contain exactly one ':'", n, got)
		}
		parts := strings.Split(got, ":")
		if len(parts) != 2 {
			t.Errorf("pi_digit(%d) split on ':' = %v", n, parts)
			continue
		}
		// First part: requested n echoed back as decimal ASCII.
		// Use fmt-style int parsing to keep this independent.
		var gotN int64
		for _, ch := range parts[0] {
			if ch < '0' || ch > '9' {
				t.Errorf("pi_digit(%d) echoed-n %q has non-digit", n, parts[0])
				break
			}
			gotN = gotN*10 + int64(ch-'0')
		}
		if gotN != n {
			t.Errorf("pi_digit(%d) echoed n = %d, want %d", n, gotN, n)
		}
		// Second part: exactly one digit.
		if len(parts[1]) != 1 || parts[1][0] < '0' || parts[1][0] > '9' {
			t.Errorf("pi_digit(%d) DIGIT part = %q, want one ASCII digit", n, parts[1])
		}
	}
}

func TestPiDigit_RejectsZero(t *testing.T) {
	_, err := runPiDigit(t, 0)
	if err == nil {
		t.Fatal("pi_digit(0) should error")
	}
	if !strings.Contains(err.Text, "n >= 1") {
		t.Errorf("pi_digit(0) error message = %q, expected mention of 'n >= 1'", err.Text)
	}
}

func TestPiDigit_RejectsNegative(t *testing.T) {
	_, err := runPiDigit(t, -1)
	if err == nil {
		t.Fatal("pi_digit(-1) should error")
	}
}

func TestPiDigit_RejectsBeyondCap(t *testing.T) {
	_, err := runPiDigit(t, piMaxDigits+1)
	if err == nil {
		t.Fatalf("pi_digit(%d) should error", piMaxDigits+1)
	}
	if !strings.Contains(err.Text, "supports n up to") {
		t.Errorf("pi_digit(%d) error message = %q, expected mention of cap", piMaxDigits+1, err.Text)
	}
}

func TestPiDigit_AtCapBoundary(t *testing.T) {
	got, err := runPiDigit(t, piMaxDigits)
	if err != nil {
		t.Fatalf("pi_digit(%d) errored: %s", piMaxDigits, err.Text)
	}
	wantPrefix := fmt.Sprintf("%d:", piMaxDigits)
	if !strings.HasPrefix(got, wantPrefix) {
		t.Errorf("pi_digit(%d) = %q, expected prefix %q", piMaxDigits, got, wantPrefix)
	}
}

func TestPiDigits_BasicSequences(t *testing.T) {
	cases := []struct {
		count int64
		want  string
	}{
		{0, ""},
		{1, "1"},
		{10, "1415926535"},
		{20, "14159265358979323846"},
	}
	for _, c := range cases {
		got, err := runPiDigits(t, c.count)
		if err != nil {
			t.Errorf("pi_digits(%d) errored: %s", c.count, err.Text)
			continue
		}
		if got != c.want {
			t.Errorf("pi_digits(%d) = %q, want %q", c.count, got, c.want)
		}
	}
}

func TestPiDigits_FullCapBoundary(t *testing.T) {
	got, err := runPiDigits(t, piMaxDigits)
	if err != nil {
		t.Fatalf("pi_digits(%d) errored: %s", piMaxDigits, err.Text)
	}
	if int64(len(got)) != piMaxDigits {
		t.Errorf("pi_digits(%d) returned length %d, want %d", piMaxDigits, len(got), piMaxDigits)
	}
}

func TestPiDigits_RejectsNegative(t *testing.T) {
	_, err := runPiDigits(t, -1)
	if err == nil {
		t.Fatal("pi_digits(-1) should error")
	}
}

func TestPiDigits_RejectsBeyondCap(t *testing.T) {
	_, err := runPiDigits(t, piMaxDigits+1)
	if err == nil {
		t.Fatalf("pi_digits(%d) should error", piMaxDigits+1)
	}
}

// TestPiDigit_AgreesWithPiDigits is a consistency check: for any n in range,
// pi_digit(n) should report the same character that pi_digits(n) ends in.
func TestPiDigit_AgreesWithPiDigits(t *testing.T) {
	for _, n := range []int64{1, 10, 100, 1000, 10_000, 100_000} {
		bulk, err := runPiDigits(t, n)
		if err != nil {
			t.Errorf("pi_digits(%d) errored: %s", n, err.Text)
			continue
		}
		single, err := runPiDigit(t, n)
		if err != nil {
			t.Errorf("pi_digit(%d) errored: %s", n, err.Text)
			continue
		}
		// Compare the digit value.
		// Single is "n:digit" — the digit is the last char.
		got := single[len(single)-1]
		want := bulk[len(bulk)-1]
		if got != want {
			t.Errorf("pi_digit(%d) trailing digit %q != pi_digits(%d) last char %q", n, got, n, want)
		}
	}
}
