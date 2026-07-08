package backstage

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/jsolana/backctl/internal/client"
)

type SearchService struct {
	client *client.Client
}

func NewSearchService(c *client.Client) *SearchService {
	return &SearchService{client: c}
}

type SearchOptions struct {
	Term    string
	Types   []string
	Filters map[string]string
	Limit   int
	Cursor  string
}

type SearchResultSummary struct {
	Ref        string `json:"ref"`
	ResultType string `json:"resultType,omitempty"`
	Title      string `json:"title"`
	Kind       string `json:"kind,omitempty"`
	Type       string `json:"type,omitempty"`
	Owner      string `json:"owner,omitempty"`
	Lifecycle  string `json:"lifecycle,omitempty"`
	Location   string `json:"location,omitempty"`
	Snippet    string `json:"snippet,omitempty"`
}

type SearchOutput struct {
	Results      []SearchResultSummary `json:"results"`
	TotalResults *int                  `json:"totalResults,omitempty"`
	NextCursor   string                `json:"nextCursor,omitempty"`
}

func (s *SearchService) Query(ctx context.Context, opts SearchOptions) (*SearchOutput, error) {
	q := url.Values{}
	q.Set("term", opts.Term)
	for _, t := range opts.Types {
		q.Add("types[]", t)
	}
	for k, v := range opts.Filters {
		q.Add(fmt.Sprintf("filters[%s]", k), v)
	}
	if opts.Limit > 0 {
		q.Set("pageLimit", fmt.Sprintf("%d", opts.Limit))
	}
	if opts.Cursor != "" {
		q.Set("pageCursor", opts.Cursor)
	}

	var raw SearchResponse
	err := s.client.GetJSON(ctx, "/api/search/query", q, &raw)
	if err != nil {
		return nil, err
	}

	output := &SearchOutput{
		TotalResults: raw.NumberOfResults,
		NextCursor:   raw.NextPageCursor,
	}

	for _, r := range raw.Results {
		ref := buildRef(r.Document)
		snippet := r.Document.Text
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}

		output.Results = append(output.Results, SearchResultSummary{
			Ref:        ref,
			ResultType: r.Type,
			Title:      r.Document.Title,
			Kind:       r.Document.Kind,
			Type:       r.Document.Type,
			Owner:      r.Document.Owner,
			Lifecycle:  r.Document.Lifecycle,
			Location:   extractDocsPath(r.Document.Location, r.Document.Kind, r.Document.Namespace, r.Document.Name),
			Snippet:    snippet,
		})
	}

	return output, nil
}

func buildRef(doc SearchDocument) string {
	if doc.Kind == "" || doc.Name == "" {
		return doc.Location
	}
	ns := doc.Namespace
	if ns == "" {
		ns = "default"
	}
	return fmt.Sprintf("%s:%s/%s", doc.Kind, ns, doc.Name)
}

func extractDocsPath(location, kind, namespace, name string) string {
	if kind == "" || name == "" {
		return ""
	}
	ns := namespace
	if ns == "" {
		ns = "default"
	}
	prefix := fmt.Sprintf("/docs/%s/%s/%s/", ns, strings.ToLower(kind), name)
	lower := strings.ToLower(location)
	idx := strings.Index(lower, strings.ToLower(prefix))
	if idx == -1 {
		return ""
	}
	return strings.TrimRight(location[idx+len(prefix):], "/")
}
