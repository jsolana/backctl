package entityref

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		input     string
		defKind   string
		defNs     string
		want      Ref
		wantErr   bool
	}{
		// Full refs
		{"component:default/my-service", "", "default", Ref{"Component", "default", "my-service"}, false},
		{"api:production/payment-api", "", "default", Ref{"API", "production", "payment-api"}, false},

		// Kind + name (no namespace)
		{"component:my-service", "", "default", Ref{"Component", "default", "my-service"}, false},
		{"api:my-api", "", "custom", Ref{"API", "custom", "my-api"}, false},

		// Name only (no kind, no namespace)
		{"my-service", "Component", "default", Ref{"Component", "default", "my-service"}, false},
		{"my-service", "", "default", Ref{"", "default", "my-service"}, false},

		// Namespace/name (no kind)
		{"production/my-service", "Component", "default", Ref{"Component", "production", "my-service"}, false},
		{"production/my-service", "", "default", Ref{"", "production", "my-service"}, false},

		// Custom kinds
		{"CustomKind:ns/name", "", "default", Ref{"CustomKind", "ns", "name"}, false},
		{"Template:default/scaffold", "", "default", Ref{"Template", "default", "scaffold"}, false},

		// Case normalization
		{"API:default/test", "", "default", Ref{"API", "default", "test"}, false},
		{"COMPONENT:my-svc", "", "default", Ref{"Component", "default", "my-svc"}, false},

		// Errors
		{"", "", "default", Ref{}, true},
		{"kind:", "", "default", Ref{}, true},
		{"kind:ns/", "", "default", Ref{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input, tt.defKind, tt.defNs)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseStrict(t *testing.T) {
	tests := []struct {
		input   string
		defNs   string
		want    Ref
		wantErr bool
	}{
		{"component:default/my-service", "default", Ref{"Component", "default", "my-service"}, false},
		{"component:my-service", "default", Ref{"Component", "default", "my-service"}, false},
		{"API:production/payment-api", "default", Ref{"API", "production", "payment-api"}, false},
		{"api:my-api", "custom", Ref{"API", "custom", "my-api"}, false},
		{"system:default/payments", "default", Ref{"System", "default", "payments"}, false},
		{"CustomKind:ns/name", "default", Ref{"CustomKind", "ns", "name"}, false},

		// Errors: missing kind
		{"", "default", Ref{}, true},
		{"my-service", "default", Ref{}, true},
		{"default/my-service", "default", Ref{}, true},
		{"nocolon", "default", Ref{}, true},
		{"kind:", "default", Ref{}, true},
		{":name", "default", Ref{}, true},
		{"kind:ns/", "default", Ref{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseStrict(tt.input, tt.defNs)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseStrict(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseStrict(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseStrict(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRef_String(t *testing.T) {
	tests := []struct {
		ref  Ref
		want string
	}{
		{Ref{"Component", "default", "my-service"}, "Component:default/my-service"},
		{Ref{"", "default", "my-service"}, "default/my-service"},
		{Ref{"", "", "my-service"}, "my-service"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.ref.String(); got != tt.want {
				t.Errorf("Ref.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRef_IsComplete(t *testing.T) {
	tests := []struct {
		ref  Ref
		want bool
	}{
		{Ref{"Component", "default", "svc"}, true},
		{Ref{"", "default", "svc"}, false},
		{Ref{"Component", "", "svc"}, false},
		{Ref{"Component", "default", ""}, false},
	}

	for _, tt := range tests {
		t.Run(tt.ref.String(), func(t *testing.T) {
			if got := tt.ref.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}
