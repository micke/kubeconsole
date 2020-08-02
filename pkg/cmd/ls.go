package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
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
		fmt.Println("ls called")
	},
	Args: func(cmd *cobra.Command, args []string) error {
		argLength := len(args)
		if argLength < 1 {
			return errors.New("requires a environment argument")
		}

		// If no context with the specified name is found
		if K8sClient.Contexts[args[0]] == nil {
			return fmt.Errorf("invalid environment specified: %s, available environments are %v", args[0], K8sClient.ContextNames())
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

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// lsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// lsCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
