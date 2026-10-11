package httpapi

import "testing"

func TestNormalizeEmail(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"User@Example.com", "user@example.com", true},
		{"  a@b.co  ", "a@b.co", true},
		{"", "", false},
		{"not-an-email", "", false},
		{"Name <a@b.co>", "", false},
		{"a@b.co, c@d.co", "", false},
	}
	for _, c := range cases {
		got, ok := normalizeEmail(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("normalizeEmail(%q) = %q,%v want %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	if validatePassword("short") == "" {
		t.Error("short password accepted")
	}
	if validatePassword("exactly12chr") != "" {
		t.Error("12-char password rejected")
	}
	long := make([]byte, 129)
	for i := range long {
		long[i] = 'a'
	}
	if validatePassword(string(long)) == "" {
		t.Error("129-char password accepted")
	}
}
