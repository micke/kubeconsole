package cmd

import (
	"errors"
	"fmt"

	"github.com/micke/kubeconsole/pkg/console"
	"github.com/spf13/cobra"
)

var (
	allEnvironments bool
	everyone        bool
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
		K8sClient.SelectContext(args[0])

		console.List(K8sClient, allEnvironments, everyone, MachineID)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if !allEnvironments {
			if len(args) < 1 {
				return errors.New("requires a environment argument")
			}

			// If no context with the specified name is found
			if K8sClient.Contexts[args[0]] == nil {
				return fmt.Errorf("invalid environment specified: %s, available environments are %v", args[0], K8sClient.ContextNames())
			}
		}

		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		// Completing context names
		case 0:
			return K8sClient.ContextNamesWithPrefix(toComplete), cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	},
}

func init() {
	rootCmd.AddCommand(lsCmd)

	lsCmd.Flags().BoolVarP(&allEnvironments, "all-environments", "a", false, "Find console pods in all environments")
	lsCmd.Flags().BoolVarP(&everyone, "everyone", "e", false, "Find everyone's console pods, not just your own console pods")
}
