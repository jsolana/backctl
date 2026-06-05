package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jsolana/backctl/internal/backstage"
	"github.com/jsolana/backctl/internal/client"
	"github.com/jsolana/backctl/internal/config"
	"github.com/jsolana/backctl/internal/output"
	"github.com/jsolana/backctl/internal/resolver"
)

type globalFlags struct {
	baseURL    string
	tokenFile  string
	namespace  string
	output     string
	noAuth     bool
	timeout    string
	verbose    bool
	configFile string
}

var flags globalFlags

func NewRootCommand(version, commit, date string) *cobra.Command {
	root := &cobra.Command{
		Use:   "backctl",
		Short: "CLI for interacting with Backstage APIs",
		Long: `backctl is a command-line tool for querying and managing Backstage catalog entities,
searching documentation, and resolving service dependencies.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return applyConfig(cmd)
		},
	}

	pf := root.PersistentFlags()
	pf.StringVar(&flags.configFile, "config", "", "config file path (default ~/.config/backctl/config.yaml)")
	pf.StringVar(&flags.baseURL, "base-url", "", "Backstage instance URL")
	pf.StringVar(&flags.tokenFile, "token-file", "", "path to file containing bearer token")
	pf.StringVarP(&flags.namespace, "namespace", "n", "", "default namespace")
	pf.StringVarP(&flags.output, "output", "o", "", "output format: table, json, yaml")
	pf.BoolVar(&flags.noAuth, "no-auth", false, "allow unauthenticated access")
	pf.StringVar(&flags.timeout, "timeout", "", "request timeout")
	pf.BoolVarP(&flags.verbose, "verbose", "v", false, "verbose output to stderr")

	root.AddCommand(
		newCatalogCmd(),
		newSearchCmd(),
		newTechDocsCmd(),
		newRelationsCmd(),
		newLocationsCmd(),
		newCompletionCmd(),
		newVersionCmd(version, commit, date),
	)

	return root
}

// applyConfig loads the config file and fills in unset flags.
// Precedence: explicit flag > env var > config file > hardcoded default.
func applyConfig(cmd *cobra.Command) error {
	cfg, err := config.Load(flags.configFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	resolve := func(flagName string, flagVal *string, envKey, cfgVal, hardDefault string) {
		if cmd.Flags().Changed(flagName) {
			return
		}
		if envVal := os.Getenv(envKey); envVal != "" {
			*flagVal = envVal
			return
		}
		if cfgVal != "" {
			*flagVal = cfgVal
			return
		}
		*flagVal = hardDefault
	}

	resolve("base-url", &flags.baseURL, "BACKSTAGE_URL", cfg.BaseURL, "")
	resolve("token-file", &flags.tokenFile, "", cfg.TokenFile, "")
	resolve("namespace", &flags.namespace, "BACKSTAGE_NAMESPACE", cfg.Namespace, "default")
	resolve("timeout", &flags.timeout, "", cfg.Timeout, "30s")
	resolve("output", &flags.output, "", cfg.Output, "")

	if !cmd.Flags().Changed("no-auth") && cfg.NoAuth {
		flags.noAuth = true
	}
	if !cmd.Flags().Changed("verbose") && cfg.Verbose {
		flags.verbose = true
	}

	return nil
}

func newClient() (*client.Client, error) {
	if flags.baseURL == "" {
		return nil, fmt.Errorf("missing Backstage URL.\n\n  Set BACKSTAGE_URL environment variable, use --base-url flag, or set base_url in config file")
	}

	token, err := resolveToken()
	if err != nil {
		return nil, err
	}

	timeout, err := time.ParseDuration(flags.timeout)
	if err != nil {
		return nil, fmt.Errorf("invalid timeout %q: %w", flags.timeout, err)
	}

	return client.New(client.Config{
		BaseURL:   flags.baseURL,
		Token:     token,
		Timeout:   timeout,
		UserAgent: "backctl",
	})
}

func resolveToken() (string, error) {
	if flags.tokenFile != "" {
		data, err := os.ReadFile(flags.tokenFile)
		if err != nil {
			return "", fmt.Errorf("cannot read token file: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}

	if token := os.Getenv("BACKSTAGE_TOKEN"); token != "" {
		return token, nil
	}

	if flags.noAuth {
		return "", nil
	}

	return "", fmt.Errorf("missing authentication token.\n\n  Options:\n    Set BACKSTAGE_TOKEN environment variable\n    Use --token-file <path> to read token from a file\n    Use --no-auth for unauthenticated access\n    Set token_file in config file (~/.config/backctl/config.yaml)\n\n  Example: BACKSTAGE_TOKEN=<your-token> backctl catalog list")
}

func newServices() (*backstage.CatalogService, *backstage.SearchService, *backstage.TechDocsService, *backstage.LocationsService, error) {
	c, err := newClient()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return backstage.NewCatalogService(c),
		backstage.NewSearchService(c),
		backstage.NewTechDocsService(c),
		backstage.NewLocationsService(c),
		nil
}

func outputFormat() output.Format {
	return output.DetectFormat(flags.output)
}

func newResolver(catalog *backstage.CatalogService) *resolver.Resolver {
	return resolver.New(catalog)
}
