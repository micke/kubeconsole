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

var options console.Options

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubeconsole [environment]",
	Short: "kubeconsole is used to create a temporary pod based on a deployments specification",
	Example: `
		# Select a deployment in the production environment
		kubeconsole production
		# Run a custom command instead of the command specified in the deployment
		kubeconsole production -- /bin/bash
	`,
	Run: func(cmd *cobra.Command, args []string) {
		k8sClient.SelectContext(args[0])
		if argsLenAtDash := cmd.ArgsLenAtDash(); argsLenAtDash > 0 {
			options.Command = args[argsLenAtDash:]
		}

		console.Start(k8sClient, options)
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
	rootCmd.PersistentFlags().StringVarP(&options.LabelSelector, "selector", "l", "process in (console, rails-shell)", "Label selector used to filter the deployments, works the same as the -l flag for kubectl")
	rootCmd.PersistentFlags().IntVar(&options.Lifetime, "lifetime", 1, "Lifetime in hours that the pod should be able to live after the heartbeat has stopped")
	rootCmd.PersistentFlags().StringVar(&options.Limits, "limits", "", "The resource requirement limits for this container. For example, 'cpu=200m,memory=512Mi'. The specified limits will also be set as requests")
	rootCmd.PersistentFlags().StringVar(&options.Image, "image", "", "The image for the container to run. Replaces the image specified in the deployment")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose")
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
