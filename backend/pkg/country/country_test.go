package country

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Brasil", "BR"},
		{"brasil", "BR"},
		{"BRAZIL", "BR"},
		{"brazil", "BR"},
		{"  brasil  ", "BR"},
		{"br", "BR"},
		{"BR", "BR"},
		{"", ""},
		{"C-1234", "C-1234"},
		{"United States", "United States"},
		{"US", "US"},
	}
	for _, tt := range tests {
		if got := Normalize(tt.in); got != tt.want {
			t.Errorf("Normalize(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestEqual(t *testing.T) {
	pairs := []struct {
		a, b string
		want bool
	}{
		{"Brasil", "BR", true},
		{"br", "BR", true},
		{"BR", "US", false},
		{"", "", true},
		{"C-1234", "C-1234", true},
		{"C-1234", "C-9999", false},
	}
	for _, p := range pairs {
		if got := Equal(p.a, p.b); got != p.want {
			t.Errorf("Equal(%q,%q) = %v, want %v", p.a, p.b, got, p.want)
		}
	}
}
