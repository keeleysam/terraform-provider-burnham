package oel

import "testing"

func TestFormat(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{`user.department  ==  'Sales'`, `user.department=="Sales"`},
		{`String.stringContains( user.dept ,  "x" )`, `String.stringContains(user.dept, "x")`},
		{`user.getInternalProperty('status')`, `user.getInternalProperty("status")`},
		{`user.isMemberOf({'group.id': '00g'})`, `user.isMemberOf({"group.id": "00g"})`},
		{`a  AND  b  AND  c`, `a AND b AND c`},
	}
	for _, tc := range cases {
		got, err := Format(tc.in)
		if err != nil {
			t.Fatalf("Format(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("Format(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatRejectsInvalid(t *testing.T) {
	for _, in := range []string{`a &&`, `String.(`, `user.`, ``, `)(`} {
		if _, err := Format(in); err == nil {
			t.Errorf("Format(%q) = nil error, want parse error", in)
		}
	}
}

func TestIsValid(t *testing.T) {
	for _, ok := range []string{
		`user.department=="Sales"`,
		`isMemberOfGroupName("x")`,
		`user.getInternalProperty("status")`,
		`user.isMemberOf({'group.id': '00g'})`,
		`user.getGroups({'group.profile.name': 'Everyone'}).![profile.name]`,
	} {
		if !IsValid(ok) {
			t.Errorf("IsValid(%q) = false, want true", ok)
		}
	}
	for _, bad := range []string{`a &&`, `foo(`, ``, `)(`, `user.`} {
		if IsValid(bad) {
			t.Errorf("IsValid(%q) = true, want false", bad)
		}
	}
}
