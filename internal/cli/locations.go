package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/jsolana/backctl/internal/output"
)

func newLocationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "locations",
		Short: "Location management",
	}
	cmd.AddCommand(
		newLocationsListCmd(),
		newLocationsGetCmd(),
	)
	return cmd
}

func newLocationsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all locations",
		Example: `  backctl locations list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, locations, err := newServices()
			if err != nil {
				return err
			}

			items, err := locations.List(cmd.Context())
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatTable {
				td := output.TableData{
					Headers: []string{"ID", "TYPE", "TARGET"},
				}
				for _, loc := range items {
					td.Rows = append(td.Rows, []string{loc.ID, loc.Type, loc.Target})
				}
				return output.Print(os.Stdout, format, td)
			}
			return output.Print(os.Stdout, format, items)
		},
	}
	return cmd
}

func newLocationsGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <id>",
		Short:   "Get location by ID",
		Example: `  backctl locations get abc-123`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, _, _, locations, err := newServices()
			if err != nil {
				return err
			}

			loc, err := locations.GetByID(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			return output.Print(os.Stdout, outputFormat(), loc)
		},
	}
	return cmd
}
