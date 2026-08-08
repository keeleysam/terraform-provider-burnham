package dataformat

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateOptionKeys(t *testing.T) {
	for _, tc := range []struct {
		name    string
		keys    []string
		allowed []string
		wantErr string
	}{
		{
			name:    "all known",
			keys:    []string{"indent", "escape_html"},
			allowed: []string{"indent", "escape_html"},
		},
		{
			name:    "empty options object",
			keys:    nil,
			allowed: []string{"indent"},
		},
		{
			name:    "single typo",
			keys:    []string{"indnt"},
			allowed: []string{"indent", "escape_html"},
			wantErr: `unsupported option "indnt"; supported options are "escape_html", "indent"`,
		},
		{
			name:    "several unknown, reported sorted",
			keys:    []string{"zzz", "aaa"},
			allowed: []string{"indent"},
			wantErr: `unsupported options "aaa", "zzz"; supported options are "indent"`,
		},
		{
			name:    "known and unknown mixed",
			keys:    []string{"indent", "compact"},
			allowed: []string{"indent", "escape_html"},
			wantErr: `unsupported option "compact"; supported options are "escape_html", "indent"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			attrs := map[string]attr.Value{}
			for _, k := range tc.keys {
				attrs[k] = types.StringValue("")
			}

			err := validateOptionKeys(attrs, tc.allowed...)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error %q, got nil", tc.wantErr)
			}
			if err.Error() != tc.wantErr {
				t.Errorf("got:  %s\nwant: %s", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestHuJSONEncode_RejectsUnknownOption is the behavior a caller actually sees:
// before this change a typo returned default-formatted output with no error,
// which is indistinguishable from an option that silently does nothing.
func TestHuJSONEncode_RejectsUnknownOption(t *testing.T) {
	obj := types.ObjectValueMust(
		map[string]attr.Type{"a": types.StringType},
		map[string]attr.Value{"a": types.StringValue("x")},
	)

	_, err := runHuJSONEncode(t, obj, makeBoolOpts("escape_htm", true))
	if err == nil {
		t.Fatal("expected an error for the misspelled option, got none")
	}
	if !strings.Contains(err.Error(), `unsupported option "escape_htm"`) {
		t.Errorf("error does not name the bad key: %v", err)
	}
	if !strings.Contains(err.Error(), "escape_html") {
		t.Errorf("error does not list the supported keys: %v", err)
	}
}
