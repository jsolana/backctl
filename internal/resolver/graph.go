package resolver

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/jsolana/backctl/internal/backstage"
	"github.com/jsolana/backctl/internal/entityref"
	"github.com/jsolana/backctl/internal/output"
)

type Options struct {
	Depth       int
	Types       []string
	TargetKind  string
	Direction   string // "outbound", "inbound", "both"
	Concurrency int
}

func (o *Options) defaults() {
	if o.Depth <= 0 {
		o.Depth = 3
	}
	if o.Direction == "" {
		o.Direction = "outbound"
	}
	if o.Concurrency <= 0 {
		o.Concurrency = 5
	}
}

type Resolver struct {
	catalog *backstage.CatalogService
}

func New(catalog *backstage.CatalogService) *Resolver {
	return &Resolver{catalog: catalog}
}

func (r *Resolver) Resolve(ctx context.Context, kind, namespace, name string, opts Options) (*output.TreeNode, error) {
	opts.defaults()

	rootRef := fmt.Sprintf("%s:%s/%s", kind, namespace, name)
	root := &output.TreeNode{Ref: rootRef, Label: rootRef}

	visited := &sync.Map{}
	visited.Store(rootRef, true)

	entity, err := r.catalog.GetEntityByName(ctx, kind, namespace, name)
	if err != nil {
		return root, nil
	}

	root.Kind = entity.Kind
	if owner, ok := entity.Spec["owner"].(string); ok {
		root.Owner = owner
	}
	if lifecycle, ok := entity.Spec["lifecycle"].(string); ok {
		root.Lifecycle = lifecycle
	}
	if tier, ok := entity.Metadata.Labels["tier"]; ok {
		root.Tier = tier
	}

	r.resolveChildren(ctx, entity, root, visited, 1, opts)
	return root, nil
}

func (r *Resolver) resolveChildren(ctx context.Context, entity *backstage.Entity, node *output.TreeNode, visited *sync.Map, depth int, opts Options) {
	if depth > opts.Depth {
		return
	}

	select {
	case <-ctx.Done():
		return
	default:
	}

	relations := filterRelations(entity.Relations, opts)

	sem := make(chan struct{}, opts.Concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, rel := range relations {
		wg.Add(1)
		go func(rel backstage.EntityRelation) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			ref := rel.TargetRef
			if _, loaded := visited.LoadOrStore(ref, true); loaded {
				return
			}

			child := &output.TreeNode{
				Ref:          ref,
				RelationType: rel.Type,
			}

			parts := parseTargetRef(ref)
			if parts == nil {
				child.Label = fmt.Sprintf("[%s] %s", rel.Type, ref)
				mu.Lock()
				node.Children = append(node.Children, child)
				mu.Unlock()
				return
			}

			childEntity, err := r.catalog.GetEntityByName(ctx, parts[0], parts[1], parts[2])
			if err != nil {
				child.Label = fmt.Sprintf("[%s] %s [unresolved]", rel.Type, ref)
				mu.Lock()
				node.Children = append(node.Children, child)
				mu.Unlock()
				return
			}

			child.Kind = childEntity.Kind
			if owner, ok := childEntity.Spec["owner"].(string); ok {
				child.Owner = owner
			}
			if lifecycle, ok := childEntity.Spec["lifecycle"].(string); ok {
				child.Lifecycle = lifecycle
			}
			if tier, ok := childEntity.Metadata.Labels["tier"]; ok {
				child.Tier = tier
			}
			child.Label = formatNodeLabel(rel.Type, ref, child.Owner, child.Lifecycle)

			mu.Lock()
			node.Children = append(node.Children, child)
			mu.Unlock()

			r.resolveChildren(ctx, childEntity, child, visited, depth+1, opts)
		}(rel)
	}
	wg.Wait()
}

func formatNodeLabel(relType, ref, owner, lifecycle string) string {
	parts := []string{fmt.Sprintf("[%s] %s", relType, ref)}
	var meta []string
	if owner != "" {
		meta = append(meta, "owner="+owner)
	}
	if lifecycle != "" {
		meta = append(meta, "lifecycle="+lifecycle)
	}
	if len(meta) > 0 {
		parts = append(parts, "("+strings.Join(meta, ", ")+")")
	}
	return strings.Join(parts, " ")
}

func filterRelations(relations []backstage.EntityRelation, opts Options) []backstage.EntityRelation {
	var result []backstage.EntityRelation
	for _, rel := range relations {
		if opts.Direction != "both" && !matchesDirection(rel.Type, opts.Direction) {
			continue
		}
		if len(opts.Types) > 0 && !containsType(opts.Types, rel.Type) {
			continue
		}
		if opts.TargetKind != "" {
			parts := parseTargetRef(rel.TargetRef)
			if parts != nil && !strings.EqualFold(parts[0], opts.TargetKind) {
				continue
			}
		}
		result = append(result, rel)
	}
	return result
}

var outboundRelationTypes = map[string]bool{
	"ownedby":      true,
	"dependson":    true,
	"consumesapi":  true,
	"providesapi":  true,
	"partof":       true,
	"memberof":     true,
	"subcomponentof": true,
}

var inboundRelationTypes = map[string]bool{
	"ownerof":        true,
	"dependencyof":   true,
	"apiconsumedby":  true,
	"apiprovidedby":  true,
	"haspart":        true,
	"hasmember":      true,
	"parentof":       true,
}

func matchesDirection(relType, direction string) bool {
	normalized := strings.ToLower(relType)
	switch direction {
	case "outbound":
		return outboundRelationTypes[normalized]
	case "inbound":
		return inboundRelationTypes[normalized]
	default:
		return true
	}
}

func containsType(types []string, t string) bool {
	for _, tt := range types {
		if strings.EqualFold(tt, t) {
			return true
		}
	}
	return false
}

// parseTargetRef parses an entity ref using the lenient parser,
// returning [kind, namespace, name] or nil if unparseable.
func parseTargetRef(ref string) []string {
	parsed, err := entityref.Parse(ref, "", "default")
	if err != nil {
		return nil
	}
	return []string{parsed.Kind, parsed.Namespace, parsed.Name}
}

func (r *Resolver) ResolveFlat(ctx context.Context, kind, namespace, name string, opts Options) ([]FlatRelation, error) {
	opts.defaults()

	entity, err := r.catalog.GetEntityByName(ctx, kind, namespace, name)
	if err != nil {
		return nil, err
	}

	relations := filterRelations(entity.Relations, opts)
	result := make([]FlatRelation, 0, len(relations))
	for _, rel := range relations {
		result = append(result, FlatRelation{
			Type:      rel.Type,
			TargetRef: rel.TargetRef,
		})
	}
	return result, nil
}

type FlatRelation struct {
	Type      string `json:"type"`
	TargetRef string `json:"targetRef"`
}
