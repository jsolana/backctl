package entityref

import (
	"fmt"
	"strings"
)

type Ref struct {
	Kind      string
	Namespace string
	Name      string
}

func (r Ref) String() string {
	if r.Kind == "" {
		if r.Namespace == "" {
			return r.Name
		}
		return r.Namespace + "/" + r.Name
	}
	return fmt.Sprintf("%s:%s/%s", r.Kind, r.Namespace, r.Name)
}

func (r Ref) IsComplete() bool {
	return r.Kind != "" && r.Namespace != "" && r.Name != ""
}

var knownKinds = map[string]string{
	"component": "Component",
	"api":       "API",
	"system":    "System",
	"domain":    "Domain",
	"resource":  "Resource",
	"group":     "Group",
	"user":      "User",
	"location":  "Location",
	"template":  "Template",
}

// Parse accepts all valid Backstage entity ref formats:
//   - "name"
//   - "namespace/name"
//   - "kind:name"
//   - "kind:namespace/name"
//
// Missing kind or namespace are filled with the provided defaults.
// An empty defaultKind means the caller accepts refs without kind.
func Parse(raw, defaultKind, defaultNamespace string) (Ref, error) {
	if raw == "" {
		return Ref{}, fmt.Errorf("entity ref cannot be empty")
	}

	var kind, namespace, name string

	if idx := strings.Index(raw, ":"); idx >= 0 {
		kind = normalizeKind(raw[:idx])
		rest := raw[idx+1:]
		if rest == "" {
			return Ref{}, fmt.Errorf("invalid entity ref %q: name cannot be empty", raw)
		}
		namespace, name = splitNamespaceName(rest, defaultNamespace)
	} else {
		kind = defaultKind
		namespace, name = splitNamespaceName(raw, defaultNamespace)
	}

	if name == "" {
		return Ref{}, fmt.Errorf("invalid entity ref %q: name cannot be empty", raw)
	}

	return Ref{Kind: kind, Namespace: namespace, Name: name}, nil
}

// ParseStrict requires the kind to be present in the ref (format: kind:[namespace/]name).
// Use this in CLI commands where kind is mandatory for API calls.
func ParseStrict(raw, defaultNamespace string) (Ref, error) {
	if raw == "" {
		return Ref{}, fmt.Errorf("entity ref cannot be empty")
	}

	idx := strings.Index(raw, ":")
	if idx <= 0 {
		return Ref{}, fmt.Errorf("invalid entity ref %q: expected format kind:[namespace/]name", raw)
	}

	kind := normalizeKind(raw[:idx])
	rest := raw[idx+1:]
	if rest == "" {
		return Ref{}, fmt.Errorf("invalid entity ref %q: name cannot be empty after kind", raw)
	}

	namespace, name := splitNamespaceName(rest, defaultNamespace)
	if name == "" {
		return Ref{}, fmt.Errorf("invalid entity ref %q: name cannot be empty", raw)
	}

	return Ref{Kind: kind, Namespace: namespace, Name: name}, nil
}

func splitNamespaceName(s, defaultNamespace string) (namespace, name string) {
	if idx := strings.Index(s, "/"); idx >= 0 {
		return s[:idx], s[idx+1:]
	}
	return defaultNamespace, s
}

func normalizeKind(kind string) string {
	if normalized, ok := knownKinds[strings.ToLower(kind)]; ok {
		return normalized
	}
	return kind
}
