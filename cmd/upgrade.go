package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/takutakahashi/operation-mcp/pkg/upgrade"
)

// upgradeCmd represents the upgrade command
var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade operations CLI to a new version",
	Long:  `Upgrade operations CLI to a new version. By default, upgrades to the latest version.`,
	Run: func(cmd *cobra.Command, args []string) {
		version, _ := cmd.Flags().GetString("version")
		output, _ := cmd.Flags().GetString("output")
		dryRun, _ := cmd.Flags().GetBool("dry-run")
		force, _ := cmd.Flags().GetBool("force")

		// Default owner and repo
		owner := "takutakahashi"
		repo := "operation-mcp"

		if err := upgrade.Upgrade(owner, repo, version, output, dryRun, force); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	// Add flags for the upgrade command
	upgradeCmd.Flags().StringP("version", "v", "", "Version to upgrade to (default is latest version)")
	upgradeCmd.Flags().StringP("output", "o", "", "Path where to install the binary (default is current binary location)")
	upgradeCmd.Flags().Bool("dry-run", false, "Only show available versions without upgrading")
	upgradeCmd.Flags().BoolP("force", "f", false, "Skip confirmation prompt")
}
