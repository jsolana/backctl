package output

import (
	"bytes"
	"testing"
)

func TestExtractText(t *testing.T) {
	html := []byte(`<html><head><title>Test</title></head><body>
<nav>Navigation</nav>
<header>Header content</header>
<main><article>
<h1>Getting Started</h1>
<p>This is a guide for setting up the service.</p>
<h2>Prerequisites</h2>
<p>You need Go 1.22 or later.</p>
</article></main>
<footer>Footer</footer>
<script>var x = 1;</script>
</body></html>`)

	result := ExtractText(html)

	if result == "" {
		t.Fatal("expected non-empty result")
	}

	// Should contain article content
	if !contains(result, "Getting Started") {
		t.Error("missing 'Getting Started'")
	}
	if !contains(result, "setting up the service") {
		t.Error("missing article content")
	}

	// Should NOT contain nav/header/footer/script content
	if contains(result, "Navigation") {
		t.Error("should not contain nav content")
	}
	if contains(result, "var x = 1") {
		t.Error("should not contain script content")
	}
}

func TestPrintTree(t *testing.T) {
	root := &TreeNode{
		Ref:   "component:default/my-service",
		Label: "component:default/my-service",
		Children: []*TreeNode{
			{
				Ref:          "resource:default/my-db",
				RelationType: "dependsOn",
				Label:        "[dependsOn] resource:default/my-db (owner=team-a, lifecycle=production)",
				Owner:        "team-a",
				Lifecycle:    "production",
				Children: []*TreeNode{
					{Ref: "group:default/team-a", RelationType: "ownedBy", Label: "[ownedBy] group:default/team-a"},
				},
			},
			{Ref: "api:default/payment-api", RelationType: "consumesApi", Label: "[consumesApi] api:default/payment-api"},
		},
	}

	var buf bytes.Buffer
	PrintTree(&buf, root)

	out := buf.String()
	if !contains(out, "component:default/my-service") {
		t.Error("missing root")
	}
	if !contains(out, "├──") || !contains(out, "└──") {
		t.Error("missing tree connectors")
	}
	if !contains(out, "team-a") {
		t.Error("missing enriched owner in tree output")
	}
}

func TestTreeToJSON_Enriched(t *testing.T) {
	root := &TreeNode{
		Ref:       "component:default/my-service",
		Kind:      "Component",
		Owner:     "group:default/team-payments",
		Lifecycle: "production",
		Children: []*TreeNode{
			{
				Ref:          "resource:default/payments-db",
				RelationType: "dependsOn",
				Kind:         "Resource",
				Owner:        "group:default/team-payments",
				Tier:         "critical",
			},
			{
				Ref:          "api:default/payment-api",
				RelationType: "providesApi",
				Kind:         "API",
				Owner:        "group:default/team-payments",
				Lifecycle:    "production",
			},
		},
	}

	result := TreeToJSON(root)

	if result["ref"] != "component:default/my-service" {
		t.Errorf("ref = %v", result["ref"])
	}
	if result["kind"] != "Component" {
		t.Errorf("kind = %v", result["kind"])
	}
	if result["owner"] != "group:default/team-payments" {
		t.Errorf("owner = %v", result["owner"])
	}
	if result["lifecycle"] != "production" {
		t.Errorf("lifecycle = %v", result["lifecycle"])
	}

	children, ok := result["children"].([]map[string]any)
	if !ok || len(children) != 2 {
		t.Fatalf("expected 2 children, got %v", result["children"])
	}

	dbNode := children[0]
	if dbNode["relationType"] != "dependsOn" {
		t.Errorf("child relationType = %v", dbNode["relationType"])
	}
	if dbNode["tier"] != "critical" {
		t.Errorf("child tier = %v", dbNode["tier"])
	}

	apiNode := children[1]
	if apiNode["kind"] != "API" {
		t.Errorf("api child kind = %v", apiNode["kind"])
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello world", 5); got != "hello..." {
		t.Errorf("Truncate = %q, want %q", got, "hello...")
	}
	if got := Truncate("hi", 5); got != "hi" {
		t.Errorf("Truncate short = %q, want %q", got, "hi")
	}
}

func contains(s, sub string) bool {
	return bytes.Contains([]byte(s), []byte(sub))
}
