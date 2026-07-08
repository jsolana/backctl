package mcp

import (
	"github.com/jsolana/backctl/internal/backstage"
	"github.com/jsolana/backctl/internal/client"
	"github.com/jsolana/backctl/internal/resolver"
	mcplib "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func NewServer(httpClient *client.Client, version string) *server.MCPServer {
	catalog := backstage.NewCatalogService(httpClient)
	search := backstage.NewSearchService(httpClient)
	techdocs := backstage.NewTechDocsService(httpClient)
	graph := resolver.New(catalog)

	h := &handlers{
		catalog:  catalog,
		search:   search,
		techdocs: techdocs,
		resolver: graph,
	}

	s := server.NewMCPServer(
		"backstage-mcp",
		version,
		server.WithToolCapabilities(false),
	)

	s.AddTool(searchTool(), h.handleSearch)
	s.AddTool(getEntityTool(), h.handleGetEntity)
	s.AddTool(getRelationshipsTool(), h.handleGetRelationships)
	s.AddTool(getTechDocsPageTool(), h.handleGetTechDocsPage)
	s.AddTool(listEntitiesTool(), h.handleListEntities)
	s.AddTool(listTechDocsPagesTool(), h.handleListTechDocsPages)
	s.AddTool(executeTool(), h.handleExecute)

	return s
}

func searchTool() mcplib.Tool {
	return mcplib.NewTool("search",
		mcplib.WithDescription("Search the Backstage catalog and TechDocs by free-text query. Returns lightweight summaries including a 'resultType' field ('software-catalog' or 'techdocs') and an entity 'ref'. For techdocs results, 'location' contains the page path to pass to get_techdocs_page; if 'location' is absent, the result points to the index page."),
		mcplib.WithString("query",
			mcplib.Required(),
			mcplib.Description("Free-text search term (e.g. 'journey finished event', 'payment service', 'onboarding guide')"),
		),
		mcplib.WithString("type",
			mcplib.Description("Filter results by type: 'software-catalog' for entities, 'techdocs' for documentation"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Maximum number of results to return (default: 25)"),
		),
	)
}

func getEntityTool() mcplib.Tool {
	return mcplib.NewTool("get_entity",
		mcplib.WithDescription("Get full details of a Backstage entity by its ref. Returns metadata, spec (including API definitions), relations, and status. Entity ref format: kind:[namespace/]name (e.g. 'component:default/my-service', 'api:driver-journeys/journey-events-v1')."),
		mcplib.WithString("ref",
			mcplib.Required(),
			mcplib.Description("Entity ref in format kind:[namespace/]name"),
		),
	)
}

func getRelationshipsTool() mcplib.Tool {
	return mcplib.NewTool("get_relationships",
		mcplib.WithDescription("Traverse the dependency graph starting from an entity. Returns a tree of related entities with owner, lifecycle, and tier metadata. Useful for understanding how services connect."),
		mcplib.WithString("ref",
			mcplib.Required(),
			mcplib.Description("Root entity ref in format kind:[namespace/]name"),
		),
		mcplib.WithNumber("depth",
			mcplib.Description("Maximum traversal depth (default: 3, max: 5)"),
		),
		mcplib.WithString("direction",
			mcplib.Description("Traversal direction: 'outbound' (default), 'inbound', or 'both'"),
		),
		mcplib.WithString("relation_type",
			mcplib.Description("Filter by relation type (e.g. 'dependsOn', 'consumesApi', 'providesApi', 'ownedBy')"),
		),
	)
}

func getTechDocsPageTool() mcplib.Tool {
	return mcplib.NewTool("get_techdocs_page",
		mcplib.WithDescription("Fetch the content of a TechDocs documentation page. Returns plain text extracted from the rendered HTML. Use list_techdocs_pages first to discover available pages."),
		mcplib.WithString("ref",
			mcplib.Required(),
			mcplib.Description("Entity ref that owns the docs in format kind:[namespace/]name"),
		),
		mcplib.WithString("path",
			mcplib.Description("Page path within the docs (e.g. 'getting-started', 'api/endpoints'). Defaults to the index page."),
		),
	)
}

func listEntitiesTool() mcplib.Tool {
	return mcplib.NewTool("list_entities",
		mcplib.WithDescription("List and filter entities in the Backstage catalog. Supports filtering by kind, spec fields, and other metadata. Use nextCursor from the response with the cursor parameter to paginate through large result sets."),
		mcplib.WithString("kind",
			mcplib.Description("Filter by entity kind: Component, API, System, Domain, Resource, Group, User"),
		),
		mcplib.WithString("filter",
			mcplib.Description("Additional filter expression (e.g. 'spec.type=openapi', 'spec.lifecycle=production', 'spec.owner=team-payments')"),
		),
		mcplib.WithNumber("limit",
			mcplib.Description("Maximum number of entities to return (default: 50, max: 200)"),
		),
		mcplib.WithString("cursor",
			mcplib.Description("Pagination cursor from the nextCursor field of a previous response. Omit to start from the beginning."),
		),
	)
}

func listTechDocsPagesTool() mcplib.Tool {
	return mcplib.NewTool("list_techdocs_pages",
		mcplib.WithDescription("List all available documentation pages (table of contents) for an entity. Use this to discover page paths before calling get_techdocs_page."),
		mcplib.WithString("ref",
			mcplib.Required(),
			mcplib.Description("Entity ref that owns the docs in format kind:[namespace/]name"),
		),
	)
}

func executeTool() mcplib.Tool {
	return mcplib.NewTool("execute",
		mcplib.WithDescription("Execute an arbitrary backctl CLI command. Use this for advanced queries not covered by the other tools (e.g. 'catalog facets --facet spec.type', 'catalog ancestry component:default/my-service', 'locations list'). Allowed subcommands: catalog, search, techdocs, relations, locations, version."),
		mcplib.WithString("command",
			mcplib.Required(),
			mcplib.Description("The backctl subcommand and arguments (e.g. 'catalog facets --facet spec.type')"),
		),
	)
}
