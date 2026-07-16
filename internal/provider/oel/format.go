package oel

import okta "github.com/keeleysam/okta-expression-parser"

// IsValid reports whether s is syntactically valid Okta EL.
func IsValid(s string) bool {
	if err := checkNestingDepth(s); err != nil {
		return false
	}
	_, err := okta.New().Parse(s)
	return err == nil
}

// Format parses s and returns its canonical Okta EL serialization: normalized spacing and quoting, precedence-derived parenthesization. It errors on syntactically invalid input.
func Format(s string) (string, error) {
	if err := checkNestingDepth(s); err != nil {
		return "", err
	}
	n, err := okta.New().Parse(s)
	if err != nil {
		return "", err
	}
	return n.String(), nil
}
