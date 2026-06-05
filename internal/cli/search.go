package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/jsolana/backctl/internal/backstage"
	"github.com/jsolana/backctl/internal/output"
)

func newSearchCmd() *cobra.Command {
	var (
		types   []string
		filters []string
		limit   int
		cursor  string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search across catalog and TechDocs",
		Example: `  backctl search "payment service"
  backctl search "api" --type software-catalog
  backctl search "auth" --filter "kind=Component" --limit 10`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, search, _, _, err := newServices()
			if err != nil {
				return err
			}

			filterMap := make(map[string]string)
			for _, f := range filters {
				parts := splitFilter(f)
				if len(parts) == 2 {
					filterMap[parts[0]] = parts[1]
				}
			}

			opts := backstage.SearchOptions{
				Term:    args[0],
				Types:   types,
				Filters: filterMap,
				Limit:   limit,
				Cursor:  cursor,
			}

			result, err := search.Query(cmd.Context(), opts)
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatTable {
				td := output.TableData{
					Headers: []string{"REF", "TITLE", "KIND", "OWNER"},
				}
				for _, r := range result.Results {
					td.Rows = append(td.Rows, []string{
						r.Ref, r.Title, r.Kind, r.Owner,
					})
				}
				return output.Print(os.Stdout, format, td)
			}
			return output.Print(os.Stdout, format, result)
		},
	}

	cmd.Flags().StringSliceVar(&types, "type", nil, "filter by search type (repeatable)")
	cmd.Flags().StringSliceVar(&filters, "filter", nil, "field filter as key=value (repeatable)")
	cmd.Flags().IntVar(&limit, "limit", 0, "max results per page")
	cmd.Flags().StringVar(&cursor, "cursor", "", "pagination cursor")
	return cmd
}

func splitFilter(s string) []string {
	for i, c := range s {
		if c == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
