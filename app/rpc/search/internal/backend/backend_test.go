package backend

import "testing"

func TestNormalizeName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty defaults to mysql", in: "", want: NameMySQL},
		{name: "mysql", in: "mysql", want: NameMySQL},
		{name: "engine", in: "engine", want: NameEngine},
		{name: "unknown falls back to mysql", in: "unknown", want: NameMySQL},
		{name: "case insensitive", in: " MySQL ", want: NameMySQL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeName(tt.in); got != tt.want {
				t.Fatalf("NormalizeName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestFactoryUsesMySQLUntilEngineBackendExists(t *testing.T) {
	factory := NewFactory(nil, NameEngine)
	if got := factory.ConfiguredBackend(); got != NameEngine {
		t.Fatalf("ConfiguredBackend = %q, want %q", got, NameEngine)
	}
	if got := factory.EffectiveBackend(); got != NameMySQL {
		t.Fatalf("EffectiveBackend = %q, want %q", got, NameMySQL)
	}
	if got := factory.Backend(nil).Name(); got != NameMySQL {
		t.Fatalf("Backend name = %q, want %q", got, NameMySQL)
	}
}
