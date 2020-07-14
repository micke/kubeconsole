package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/micke/kubeconsole/pkg/console"
	"github.com/micke/kubeconsole/pkg/k8s"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var config string
var kubeconfig string
var labelSelector string
var lifetime int
var verbose bool
var k8sClient *k8s.K8s

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubeconsole [environment]",
	Short: "kubeconsole is used to create a temporary pod based on a deployments specification",
	Run: func(cmd *cobra.Command, args []string) {
		k8sClient.SelectContext(args[0])
		command := []string{}
		if argsLenAtDash := cmd.ArgsLenAtDash(); argsLenAtDash > 0 {
			command = args[argsLenAtDash:]
		}

		console.Start(k8sClient, labelSelector, lifetime, command)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("requires a environment argument")
		}

		if k8sClient.Contexts[args[0]] == nil {
			return fmt.Errorf("invalid environment specified: %s, available environments are %v", args[0], k8sClient.ContextNames())
		}

		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return k8sClient.ContextNamesWithPrefix(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&config, "config", "c", "", "config file (default $HOME/.config/kubeconsole)")
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig file (default $HOME/.kube/config)")
	rootCmd.PersistentFlags().StringVarP(&labelSelector, "selector", "l", "process in (console, rails-shell)", "label selector, works the same as the -l flag for kubectl")
	rootCmd.PersistentFlags().IntVar(&lifetime, "lifetime", 1, "lifetime in hours that the pod should be able to live after the heartbeat has stopped")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose")
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if config != "" {
		// Use config file from the flag.
		viper.SetConfigFile(config)
	} else {
		// Search config in home directory with name ".config/kubeconsole" (without extension).
		viper.AddConfigPath(home + "/.config")
		viper.SetConfigName("kubeconsole")
		viper.SetConfigType("yaml")
	}

	if kubeconfig == "" {
		kubeconfig = home + "/.kube/config"
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if verbose {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}

	k8sClient = k8s.NewK8s(kubeconfig)
}
