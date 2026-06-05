package backstage

type Entity struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   EntityMeta        `json:"metadata"`
	Spec       map[string]any    `json:"spec,omitempty"`
	Relations  []EntityRelation  `json:"relations,omitempty"`
	Status     *EntityStatus     `json:"status,omitempty"`
}

type EntityMeta struct {
	UID         string            `json:"uid,omitempty"`
	Etag        string            `json:"etag,omitempty"`
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Links       []EntityLink      `json:"links,omitempty"`
}

type EntityLink struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	Icon  string `json:"icon,omitempty"`
	Type  string `json:"type,omitempty"`
}

type EntityRelation struct {
	Type      string          `json:"type"`
	TargetRef string          `json:"targetRef"`
	Target    EntityRelTarget `json:"target,omitempty"`
}

type EntityRelTarget struct {
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type EntityStatus struct {
	Items []EntityStatusItem `json:"items,omitempty"`
}

type EntityStatusItem struct {
	Type    string `json:"type"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type LocationEntry struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Target string `json:"target"`
}

type FacetsResponse struct {
	Facets map[string][]FacetValue `json:"facets"`
}

type FacetValue struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type SearchResponse struct {
	Results        []SearchResult `json:"results"`
	NextPageCursor string         `json:"nextPageCursor,omitempty"`
	PrevPageCursor string         `json:"previousPageCursor,omitempty"`
	NumberOfResults *int          `json:"numberOfResults,omitempty"`
}

type SearchResult struct {
	Type      string         `json:"type"`
	Document  SearchDocument `json:"document"`
	Highlight *Highlight     `json:"highlight,omitempty"`
	Rank      *int           `json:"rank,omitempty"`
}

type SearchDocument struct {
	Title     string `json:"title"`
	Text      string `json:"text"`
	Location  string `json:"location"`
	Kind      string `json:"kind,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
	Lifecycle string `json:"lifecycle,omitempty"`
	Owner     string `json:"owner,omitempty"`
	Type      string `json:"type,omitempty"`
}

type Highlight struct {
	PreTag       string            `json:"preTag"`
	PostTag      string            `json:"postTag"`
	Fields       map[string]string `json:"fields"`
}

type TechDocsMetadata struct {
	SiteName        string `json:"site_name"`
	SiteDescription string `json:"site_description,omitempty"`
	BuildTimestamp  string `json:"build_timestamp,omitempty"`
}

type TechDocsEntityMetadata struct {
	APIVersion  string         `json:"apiVersion"`
	Kind        string         `json:"kind"`
	Metadata    EntityMeta     `json:"metadata"`
	Spec        map[string]any `json:"spec,omitempty"`
}

type TechDocsPage struct {
	Location string `json:"location"`
	Title    string `json:"title"`
}

type AncestryResponse struct {
	RootEntityRef string         `json:"rootEntityRef"`
	Items         []AncestryItem `json:"items"`
}

type AncestryItem struct {
	Entity          Entity   `json:"entity"`
	ParentEntityRefs []string `json:"parentEntityRefs"`
}

type ValidateEntityRequest struct {
	Entity   Entity `json:"entity"`
	Location string `json:"location,omitempty"`
}

type ValidateEntityResponse struct {
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}
