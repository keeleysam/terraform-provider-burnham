package cel

import (
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"normalizes spacing", "a   &&  b", `a && b`},
		{"macro round-trips", `x.exists(y, y > 0)`, `x.exists(y, y > 0)`},
		{"normalizes quotes", `a == 'x'`, `a == "x"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Format(tc.in)
			if err != nil {
				t.Fatalf("Format(%q) error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("Format(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestValidateRejectsInvalid(t *testing.T) {
	for _, in := range []string{`a &&`, `foo(`, `1 +`} {
		if _, err := Format(in); err == nil {
			t.Fatalf("Format(%q) = nil error, want parse error", in)
		}
	}
}

func TestIsValid(t *testing.T) {
	for _, ok := range []string{
		`a && b`,
		`x.exists(y, y > 0)`,
		`msg.?field.orValue(0)`,
		`m.all(k, v, v > 0)`,
		`resource.name.startsWith("prod-")`,
	} {
		if !IsValid(ok, false) {
			t.Errorf("IsValid(%q, false) = false, want true", ok)
		}
	}
	for _, bad := range []string{`a &&`, `foo(`, `1 +`, ``, `)(`} {
		if IsValid(bad, false) {
			t.Errorf("IsValid(%q, false) = true, want false", bad)
		}
	}
	// strict rejects optional-navigation syntax that the lenient mode accepts.
	if !IsValid(`msg.?field`, false) {
		t.Errorf("lenient IsValid(optional) = false, want true")
	}
	if IsValid(`msg.?field`, true) {
		t.Errorf("strict IsValid(optional) = true, want false")
	}
	if !IsValid(`a && b`, true) {
		t.Errorf("strict IsValid(plain) = false, want true")
	}
}

// TestFormatWrapOnOperators locks that the human-friendly operator symbols the
// docs advertise (wrap_on_operators = ["&&", "||"]) actually work: cel-go's
// unparser wants operator ids like "_&&_", so the friendly symbol must be
// translated. Passing "&&" must wrap, not error.
func TestFormatWrapOnOperators(t *testing.T) {
	for _, op := range []string{"&&", "||"} {
		in := `aaaaa == 1 ` + op + ` bbbbb == 2 ` + op + ` ccccc == 3 ` + op + ` ddddd == 4`
		got, err := Format(in, WrapColumn(30), WrapOperators(op), WrapAfter(true))
		if err != nil {
			t.Fatalf("Format with wrap_on_operators=[%q] error: %v", op, err)
		}
		if !strings.Contains(got, "\n") {
			t.Fatalf("op %q: expected wrapped output with newlines, got %q", op, got)
		}
		if !strings.Contains(got, op) {
			t.Fatalf("op %q: expected operator to survive in output, got %q", op, got)
		}
	}
}

// TestFormatWrapOnOperatorsInvalid locks the documented pass-through behavior: a
// symbol the translation does not recognize reaches cel-go unchanged, which
// rejects it, so Format returns an error rather than silently ignoring it.
func TestFormatWrapOnOperatorsInvalid(t *testing.T) {
	_, err := Format(`a && b`, WrapColumn(4), WrapOperators("nonsense"))
	if err == nil {
		t.Fatal("wrap on an unknown operator symbol should error")
	}
}

func TestValidatePrettyWraps(t *testing.T) {
	in := `aaaaa == 1 && bbbbb == 2 && ccccc == 3 && ddddd == 4`
	got, err := Format(in, WrapColumn(20))
	if err != nil {
		t.Fatalf("Validate error: %v", err)
	}
	if !strings.Contains(got, "\n") {
		t.Fatalf("expected wrapped output with newlines, got %q", got)
	}
}
