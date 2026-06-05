package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jsolana/backctl/internal/backstage"
	"github.com/jsolana/backctl/internal/entityref"
	"github.com/jsolana/backctl/internal/output"
)

func newCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Catalog operations",
	}
	cmd.AddCommand(
		newCatalogListCmd(),
		newCatalogGetCmd(),
		newCatalogAncestryCmd(),
		newCatalogValidateCmd(),
		newCatalogRefreshCmd(),
		newCatalogFacetsCmd(),
	)
	return cmd
}

func newCatalogListCmd() *cobra.Command {
	var (
		kind    string
		filters []string
		limit   int
		offset  int
		after   string
		sort    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List catalog entities",
		Example: `  backctl catalog list --kind component
  backctl catalog list --filter "spec.lifecycle=production" --limit 50
  backctl catalog list --kind api --sort metadata.name
  backctl catalog list --limit 20 --after <cursor>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			catalog, _, _, _, err := newServices()
			if err != nil {
				return err
			}

			opts := backstage.ListEntitiesOptions{
				Filters: filters,
				Limit:   limit,
				Offset:  offset,
				After:   after,
			}
			if kind != "" {
				opts.Filters = append(opts.Filters, "kind="+kind)
			}
			if sort != "" {
				opts.Order = []string{sort}
			}

			result, err := catalog.ListEntities(cmd.Context(), opts)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatTable {
				td := output.TableData{
					Headers: []string{"KIND", "NAMESPACE", "NAME", "OWNER", "LIFECYCLE"},
				}
				for _, e := range result.Entities {
					owner, _ := e.Spec["owner"].(string)
					lifecycle, _ := e.Spec["lifecycle"].(string)
					td.Rows = append(td.Rows, []string{
						e.Kind, e.Metadata.Namespace, e.Metadata.Name, owner, lifecycle,
					})
				}
				if result.NextCursor != "" {
					fmt.Fprintf(os.Stderr, "\nNext page: --after %s\n", result.NextCursor)
				}
				return output.Print(os.Stdout, format, td)
			}
			return output.Print(os.Stdout, format, result)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "filter by entity kind")
	cmd.Flags().StringSliceVar(&filters, "filter", nil, "filter expressions (repeatable)")
	cmd.Flags().IntVar(&limit, "limit", 0, "max number of results")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset for pagination")
	cmd.Flags().StringVar(&after, "after", "", "cursor for next page (from previous response)")
	cmd.Flags().StringVar(&sort, "sort", "", "sort field (e.g. metadata.name)")
	return cmd
}

func newCatalogGetCmd() *cobra.Command {
	var uid string

	cmd := &cobra.Command{
		Use:   "get [entity-ref]",
		Short: "Get a catalog entity by reference",
		Long: `Retrieve the full entity including metadata, spec, relations, and status.
Entity ref format: kind:[namespace/]name`,
		Example: `  backctl catalog get component:default/my-service
  backctl catalog get api:my-api -o json
  backctl catalog get --uid abc-123-def`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			catalog, _, _, _, err := newServices()
			if err != nil {
				return err
			}

			var entity *backstage.Entity

			if uid != "" {
				entity, err = catalog.GetEntityByUID(cmd.Context(), uid)
			} else if len(args) == 1 {
				ref, parseErr := entityref.ParseStrict(args[0], flags.namespace)
				if parseErr != nil {
					return parseErr
				}
				entity, err = catalog.GetEntityByName(cmd.Context(), ref.Kind, ref.Namespace, ref.Name)
			} else {
				return fmt.Errorf("provide an entity ref or --uid")
			}

			if err != nil {
				return err
			}
			return output.Print(os.Stdout, outputFormat(), entity)
		},
	}

	cmd.Flags().StringVar(&uid, "uid", "", "get entity by UID")
	return cmd
}

func newCatalogAncestryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ancestry <entity-ref>",
		Short: "Show entity ancestry (provenance chain)",
		Long:  `Shows how an entity arrived in the catalog: the chain of locations from the root source to the entity itself.`,
		Example: `  backctl catalog ancestry component:default/my-service
  backctl catalog ancestry component:default/my-service -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := entityref.ParseStrict(args[0], flags.namespace)
			if err != nil {
				return err
			}

			catalog, _, _, _, err := newServices()
			if err != nil {
				return err
			}

			result, err := catalog.GetAncestry(cmd.Context(), ref.Kind, ref.Namespace, ref.Name)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatTable {
				td := output.TableData{
					Headers: []string{"ENTITY REF", "KIND", "PARENTS"},
				}
				for _, item := range result.Items {
					entityRef := fmt.Sprintf("%s:%s/%s",
						item.Entity.Kind, item.Entity.Metadata.Namespace, item.Entity.Metadata.Name)
					parents := ""
					if len(item.ParentEntityRefs) > 0 {
						parents = item.ParentEntityRefs[0]
						if len(item.ParentEntityRefs) > 1 {
							parents += fmt.Sprintf(" (+%d)", len(item.ParentEntityRefs)-1)
						}
					}
					td.Rows = append(td.Rows, []string{entityRef, item.Entity.Kind, parents})
				}
				return output.Print(os.Stdout, format, td)
			}
			return output.Print(os.Stdout, format, result)
		},
	}
	return cmd
}

func newCatalogValidateCmd() *cobra.Command {
	var (
		file     string
		location string
	)

	cmd := &cobra.Command{
		Use:   "validate -f <file>",
		Short: "Validate an entity definition without registering it",
		Long:  `Performs a dry-run validation of an entity YAML/JSON file against the Backstage catalog schema.`,
		Example: `  backctl catalog validate -f catalog-info.yaml
  backctl catalog validate -f - < catalog-info.yaml
  backctl catalog validate -f catalog-info.yaml --location url:https://github.com/org/repo/blob/main/catalog-info.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if file == "" {
				return fmt.Errorf("--file (-f) is required")
			}

			var data []byte
			var err error
			if file == "-" {
				data, err = io.ReadAll(os.Stdin)
			} else {
				data, err = os.ReadFile(file)
			}
			if err != nil {
				return fmt.Errorf("reading entity file: %w", err)
			}

			var entity backstage.Entity
			if err := yaml.Unmarshal(data, &entity); err != nil {
				return fmt.Errorf("parsing entity: %w", err)
			}

			catalog, _, _, _, err := newServices()
			if err != nil {
				return err
			}

			result, err := catalog.ValidateEntity(cmd.Context(), entity, location)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatTable {
				if result.Valid {
					fmt.Fprintln(os.Stdout, "Valid: true")
				} else {
					fmt.Fprintln(os.Stdout, "Valid: false")
					fmt.Fprintln(os.Stdout, "Errors:")
					for _, e := range result.Errors {
						fmt.Fprintf(os.Stdout, "  - %s\n", e)
					}
				}
				return nil
			}
			return output.Print(os.Stdout, format, result)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "path to entity YAML/JSON file (use - for stdin)")
	cmd.Flags().StringVar(&location, "location", "", "optional source location hint")
	_ = cmd.MarkFlagRequired("file")
	return cmd
}

func newCatalogRefreshCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "refresh <entity-ref>",
		Short: "Trigger immediate entity refresh",
		Long:  `Forces Backstage to re-read and re-process the entity from its source location. This is a write operation.`,
		Example: `  backctl catalog refresh component:default/my-service
  backctl catalog refresh api:default/payment-api`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := entityref.ParseStrict(args[0], flags.namespace)
			if err != nil {
				return err
			}

			catalog, _, _, _, err := newServices()
			if err != nil {
				return err
			}

			if err := catalog.RefreshEntity(cmd.Context(), ref.String()); err != nil {
				return err
			}

			fmt.Fprintf(os.Stderr, "Refresh triggered for %s\n", ref.String())
			return nil
		},
	}
	return cmd
}

func newCatalogFacetsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "facets --field <field>",
		Short: "List available values for a catalog field",
		Example: `  backctl catalog facets --field kind
  backctl catalog facets --field spec.type
  backctl catalog facets --field spec.lifecycle`,
		RunE: func(cmd *cobra.Command, args []string) error {
			field, _ := cmd.Flags().GetString("field")
			if field == "" {
				return fmt.Errorf("--field is required")
			}

			catalog, _, _, _, err := newServices()
			if err != nil {
				return err
			}

			result, err := catalog.GetFacets(cmd.Context(), field)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatTable {
				td := output.TableData{Headers: []string{"VALUE", "COUNT"}}
				for _, facets := range result.Facets {
					for _, fv := range facets {
						td.Rows = append(td.Rows, []string{fv.Value, fmt.Sprintf("%d", fv.Count)})
					}
				}
				return output.Print(os.Stdout, format, td)
			}
			return output.Print(os.Stdout, format, result)
		},
	}

	cmd.Flags().String("field", "", "facet field name")
	_ = cmd.MarkFlagRequired("field")
	return cmd
}
