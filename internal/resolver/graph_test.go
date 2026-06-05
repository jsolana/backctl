package resolver

import (
	"testing"

	"github.com/jsolana/backctl/internal/backstage"
)

func TestParseTargetRef(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"component:default/my-service", []string{"Component", "default", "my-service"}},
		{"api:production/payment-api", []string{"API", "production", "payment-api"}},
		{"resource:my-db", []string{"Resource", "default", "my-db"}},
		// Name-only refs are now valid (kind becomes empty string)
		{"my-service", []string{"", "default", "my-service"}},
		// Empty string is still invalid
		{"", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseTargetRef(tt.input)
			if tt.want == nil {
				if got != nil {
					t.Errorf("expected nil, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil result")
			}
			for i, v := range tt.want {
				if got[i] != v {
					t.Errorf("index %d: got %q, want %q", i, got[i], v)
				}
			}
		})
	}
}

func TestFilterRelations(t *testing.T) {
	relations := []backstage.EntityRelation{
		{Type: "dependsOn", TargetRef: "resource:default/my-db"},
		{Type: "consumesApi", TargetRef: "api:default/payment-api"},
		{Type: "ownedBy", TargetRef: "group:default/team-a"},
		{Type: "ownerOf", TargetRef: "component:default/my-service"},
		{Type: "dependencyOf", TargetRef: "component:default/other"},
	}

	tests := []struct {
		name      string
		opts      Options
		wantCount int
	}{
		{
			name:      "direction both returns all",
			opts:      Options{Direction: "both"},
			wantCount: 5,
		},
		{
			name:      "direction outbound filters to outbound types",
			opts:      Options{Direction: "outbound"},
			wantCount: 3,
		},
		{
			name:      "direction inbound filters to inbound types",
			opts:      Options{Direction: "inbound"},
			wantCount: 2,
		},
		{
			name:      "filter by type",
			opts:      Options{Direction: "both", Types: []string{"dependsOn"}},
			wantCount: 1,
		},
		{
			name:      "filter by target kind",
			opts:      Options{Direction: "both", TargetKind: "api"},
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterRelations(relations, tt.opts)
			if len(got) != tt.wantCount {
				t.Errorf("got %d relations, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestMatchesDirection(t *testing.T) {
	tests := []struct {
		relType   string
		direction string
		want      bool
	}{
		{"dependsOn", "outbound", true},
		{"consumesApi", "outbound", true},
		{"ownedBy", "outbound", true},
		{"providesApi", "outbound", true},
		{"ownerOf", "inbound", true},
		{"dependencyOf", "inbound", true},
		{"apiConsumedBy", "inbound", true},
		{"dependsOn", "inbound", false},
		{"ownerOf", "outbound", false},
		{"anything", "both", true},
	}

	for _, tt := range tests {
		t.Run(tt.relType+"_"+tt.direction, func(t *testing.T) {
			got := matchesDirection(tt.relType, tt.direction)
			if got != tt.want {
				t.Errorf("matchesDirection(%q, %q) = %v, want %v", tt.relType, tt.direction, got, tt.want)
			}
		})
	}
}

func TestContainsType(t *testing.T) {
	if !containsType([]string{"dependsOn", "consumesApi"}, "dependson") {
		t.Error("should match case-insensitively")
	}
	if containsType([]string{"dependsOn"}, "consumesApi") {
		t.Error("should not match different type")
	}
}
