package cli

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/jsolana/backctl/internal/entityref"
	"github.com/jsolana/backctl/internal/output"
	"github.com/jsolana/backctl/internal/resolver"
)

func newRelationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relations",
		Short: "Dependency resolution",
	}
	cmd.AddCommand(
		newRelationsTreeCmd(),
		newRelationsListCmd(),
	)
	return cmd
}

func newRelationsTreeCmd() *cobra.Command {
	var (
		depth      int
		types      []string
		targetKind string
		direction  string
	)

	cmd := &cobra.Command{
		Use:   "tree <entity-ref>",
		Short: "Show dependency tree",
		Example: `  backctl relations tree component:default/my-service
  backctl relations tree component:default/my-service --depth 5 --type dependsOn
  backctl relations tree component:default/my-service --direction inbound`,
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

			r := newResolver(catalog)
			tree, err := r.Resolve(cmd.Context(), ref.Kind, ref.Namespace, ref.Name, resolver.Options{
				Depth:      depth,
				Types:      types,
				TargetKind: targetKind,
				Direction:  direction,
			})
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatJSON {
				return output.Print(os.Stdout, format, output.TreeToJSON(tree))
			}
			output.PrintTree(os.Stdout, tree)
			return nil
		},
	}

	cmd.Flags().IntVar(&depth, "depth", 3, "max traversal depth")
	cmd.Flags().StringSliceVar(&types, "type", nil, "filter by relation type (repeatable)")
	cmd.Flags().StringVar(&targetKind, "target-kind", "", "filter by target entity kind")
	cmd.Flags().StringVar(&direction, "direction", "outbound", "relation direction: outbound, inbound, both")
	return cmd
}

func newRelationsListCmd() *cobra.Command {
	var (
		types      []string
		targetKind string
	)

	cmd := &cobra.Command{
		Use:   "list <entity-ref>",
		Short: "List relations (flat)",
		Example: `  backctl relations list component:default/my-service
  backctl relations list component:default/my-service --type dependsOn`,
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

			r := newResolver(catalog)
			relations, err := r.ResolveFlat(cmd.Context(), ref.Kind, ref.Namespace, ref.Name, resolver.Options{
				Types:      types,
				TargetKind: targetKind,
			})
			if err != nil {
				return err
			}

			format := outputFormat()
			if format == output.FormatTable {
				td := output.TableData{
					Headers: []string{"TYPE", "TARGET"},
				}
				for _, rel := range relations {
					td.Rows = append(td.Rows, []string{rel.Type, rel.TargetRef})
				}
				return output.Print(os.Stdout, format, td)
			}
			return output.Print(os.Stdout, format, relations)
		},
	}

	cmd.Flags().StringSliceVar(&types, "type", nil, "filter by relation type")
	cmd.Flags().StringVar(&targetKind, "target-kind", "", "filter by target kind")
	return cmd
}
