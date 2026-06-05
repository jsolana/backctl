package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jsolana/backctl/internal/entityref"
	"github.com/jsolana/backctl/internal/output"
)

func newTechDocsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "techdocs",
		Short: "TechDocs operations",
	}
	cmd.AddCommand(
		newTechDocsGetCmd(),
		newTechDocsListPagesCmd(),
		newTechDocsMetadataCmd(),
		newTechDocsEntityCmd(),
	)
	return cmd
}

func newTechDocsGetCmd() *cobra.Command {
	var (
		pagePath string
		raw      bool
	)

	cmd := &cobra.Command{
		Use:   "get <entity-ref>",
		Short: "Fetch TechDocs page content",
		Long:  "Fetches a TechDocs page and extracts readable text. Use --raw for original HTML.",
		Example: `  backctl techdocs get component:default/my-service
  backctl techdocs get component:default/cax --path "guides/subscriptions"
  backctl techdocs get component:default/cax --path "guides/setup" --raw`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := entityref.ParseStrict(args[0], flags.namespace)
			if err != nil {
				return err
			}

			_, _, techdocs, _, err := newServices()
			if err != nil {
				return err
			}

			content, err := techdocs.GetPage(cmd.Context(), ref.Namespace, ref.Kind, ref.Name, pagePath)
			if err != nil {
				return err
			}

			format := outputFormat()
			if raw {
				fmt.Fprint(os.Stdout, string(content))
				return nil
			}

			text := output.ExtractText(content)

			if format == output.FormatJSON {
				result := map[string]string{
					"entityRef": ref.String(),
					"path":      pagePath,
					"content":   text,
				}
				return output.Print(os.Stdout, format, result)
			}

			fmt.Fprint(os.Stdout, text)
			return nil
		},
	}

	cmd.Flags().StringVar(&pagePath, "path", "", "docs page path (default: index)")
	cmd.Flags().BoolVar(&raw, "raw", false, "output raw HTML without text extraction")
	return cmd
}

func newTechDocsListPagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-pages <entity-ref>",
		Short: "List available TechDocs pages for an entity",
		Example: `  backctl techdocs list-pages component:default/my-service
  backctl techdocs list-pages component:default/cax -o json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := entityref.ParseStrict(args[0], flags.namespace)
			if err != nil {
				return err
			}

			_, _, techdocs, _, err := newServices()
			if err != nil {
				return err
			}

			pages, err := techdocs.ListPages(cmd.Context(), ref.Namespace, ref.Kind, ref.Name)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatTable {
				td := output.TableData{
					Headers: []string{"PATH", "TITLE"},
				}
				for _, p := range pages {
					td.Rows = append(td.Rows, []string{p.Location, p.Title})
				}
				return output.Print(os.Stdout, format, td)
			}
			return output.Print(os.Stdout, format, pages)
		},
	}
	return cmd
}

func newTechDocsMetadataCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "metadata <entity-ref>",
		Short:   "Get TechDocs build/site metadata",
		Example: `  backctl techdocs metadata component:default/my-service`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := entityref.ParseStrict(args[0], flags.namespace)
			if err != nil {
				return err
			}

			_, _, techdocs, _, err := newServices()
			if err != nil {
				return err
			}

			meta, err := techdocs.GetMetadata(cmd.Context(), ref.Namespace, ref.Kind, ref.Name)
			if err != nil {
				return err
			}

			return output.Print(os.Stdout, outputFormat(), meta)
		},
	}
	return cmd
}

func newTechDocsEntityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "entity <entity-ref>",
		Short:   "Get entity metadata in TechDocs context",
		Example: `  backctl techdocs entity component:default/my-service`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ref, err := entityref.ParseStrict(args[0], flags.namespace)
			if err != nil {
				return err
			}

			_, _, techdocs, _, err := newServices()
			if err != nil {
				return err
			}

			meta, err := techdocs.GetEntityMetadata(cmd.Context(), ref.Namespace, ref.Kind, ref.Name)
			if err != nil {
				return err
			}

			return output.Print(os.Stdout, outputFormat(), meta)
		},
	}
	return cmd
}
