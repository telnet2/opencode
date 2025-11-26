package commands

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/opencode-ai/opencode/internal/config"
	"github.com/opencode-ai/opencode/internal/provider"
	"github.com/spf13/cobra"
)

var (
	modelsVerbose bool
	modelsRefresh bool
)

var modelsCmd = &cobra.Command{
	Use:   "models [provider]",
	Short: "List available models",
	Long: `List all available models from configured providers.

Examples:
  opencode models              # List all models
  opencode models anthropic    # List only Anthropic models
  opencode models --verbose    # Show pricing information`,
	RunE: runModels,
}

func init() {
	modelsCmd.Flags().BoolVarP(&modelsVerbose, "verbose", "v", false, "Include metadata like costs")
	modelsCmd.Flags().BoolVar(&modelsRefresh, "refresh", false, "Refresh models cache")
}

func runModels(cmd *cobra.Command, args []string) error {
	// Get working directory
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Initialize paths
	paths := config.GetPaths()
	if err := paths.EnsurePaths(); err != nil {
		return err
	}

	// Load configuration
	appConfig, err := config.Load(workDir)
	if err != nil {
		return err
	}

	// Initialize providers
	ctx := context.Background()
	providerReg, err := provider.InitializeProviders(ctx, appConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize providers: %w", err)
	}

	// Get provider filter
	var providerFilter string
	if len(args) > 0 {
		providerFilter = args[0]
	}

	// Get models using AllModels
	models := providerReg.AllModels()

	// Create table writer
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	if modelsVerbose {
		fmt.Fprintln(w, "PROVIDER\tMODEL\tCONTEXT\tMAX OUTPUT\tINPUT PRICE\tOUTPUT PRICE\t")
	} else {
		fmt.Fprintln(w, "PROVIDER\tMODEL\tCONTEXT\tFEATURES\t")
	}

	for _, model := range models {
		// Apply provider filter
		if providerFilter != "" && model.ProviderID != providerFilter {
			continue
		}

		if modelsVerbose {
			fmt.Fprintf(w, "%s\t%s\t%dk\t%d\t$%.2f/1M\t$%.2f/1M\t\n",
				model.ProviderID,
				model.ID,
				model.ContextLength/1000,
				model.MaxOutputTokens,
				model.InputPrice,
				model.OutputPrice,
			)
		} else {
			features := ""
			if model.SupportsVision {
				features += "vision "
			}
			if model.SupportsTools {
				features += "tools "
			}
			if model.SupportsReasoning {
				features += "reasoning "
			}
			fmt.Fprintf(w, "%s\t%s\t%dk\t%s\t\n",
				model.ProviderID,
				model.ID,
				model.ContextLength/1000,
				features,
			)
		}
	}

	return w.Flush()
}
