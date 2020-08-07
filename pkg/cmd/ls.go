package cmd

import (
	"fmt"
	"strings"

	"github.com/micke/kubeconsole/pkg/console"
	"github.com/spf13/cobra"
)

var (
	everyone bool
)

var lsCmd = &cobra.Command{
	Use:   "ls [environment]",
	Short: "Lists all the currently running console pods",
	Example: `# List your console pods in the production environment
kubeconsole ls
# List your console pods in all environments
kubeconsole ls --all-environments
# List everyones console pods in all environments
kubeconsole ls --everyone --all-environment`,
	Run: func(cmd *cobra.Command, args []string) {
		var environments []string

		if len(args) > 0 {
			environments = args
		} else {
			environments = K8sClient.ContextNames()
		}

		console.List(K8sClient, environments, everyone, MachineID)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		for _, environment := range args {
			// If no context with the specified name is found
			if K8sClient.Contexts[environment] == nil {
				return fmt.Errorf("invalid environment specified: %s, available environments are: %v", args[0], strings.Join(K8sClient.ContextNames(), ", "))
			}
		}

		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// Completing context names
		return K8sClient.ContextNamesWithPrefix(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.Flags().BoolVarP(&everyone, "everyone", "e", false, "Find everyone's console pods, not just your own console pods")
}
