package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/micke/kubeconsole/pkg/console"
	"github.com/micke/kubeconsole/pkg/k8s"
	"github.com/spf13/cobra"

	"github.com/spf13/viper"
)

var (
	// Config specifies the path to the kubeconsole config file
	Config string
	// Kubeconfig specifies the path to the kubectl config file
	Kubeconfig string
	// Verbose specifies if verbose is enabled or not
	Verbose bool
	// K8sClient is a instance of K8s that holds common kubernetes objects
	K8sClient *k8s.K8s
	// MachineID is used to match console pods to this machine
	MachineID string
	options   console.Options
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "kubeconsole [environment]",
	Short: "kubeconsole is used to create a temporary pod based on a deployments specification",
	Example: `# Select a deployment in the production environment
kubeconsole production
# Run a custom command instead of the command specified in the deployment
kubeconsole production -- /bin/bash`,
	Run: func(cmd *cobra.Command, args []string) {
		K8sClient.SelectContext(args[0])

		if argsLenAtDash := cmd.ArgsLenAtDash(); argsLenAtDash > 0 {
			options.Command = args[argsLenAtDash:]
		}

		options.MachineID = MachineID

		console.Start(K8sClient, options)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		argLength := len(args)
		if argLength < 1 {
			return errors.New("requires a environment argument")
		}

		// If no context with the specified name is found
		if K8sClient.Contexts[args[0]] == nil {
			return fmt.Errorf("invalid environment specified: %s, available environments are: %v", args[0], strings.Join(K8sClient.ContextNames(), ", "))
		}

		// If there is a second argument that's not dashes then we assign it to DeploymentName
		if argLength > 1 && cmd.ArgsLenAtDash() != 1 {
			options.DeploymentName = args[1]
		} else {
			// Otherwise we set DeploymentName to the current path
			path, err := os.Getwd()
			if err == nil {
				options.DeploymentName = filepath.Base(path)
			}
		}

		return nil
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		switch len(args) {
		// Completing context names
		case 0:
			return K8sClient.ContextNamesWithPrefix(toComplete), cobra.ShellCompDirectiveNoFileComp
		// Completing deployment names
		case 1:
			K8sClient.SelectContext(args[0])
			return K8sClient.DeploymentNamesWithPrefix(toComplete, options.LabelSelector), cobra.ShellCompDirectiveNoFileComp
		default:
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
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

	rootCmd.PersistentFlags().StringVarP(&Config, "config", "c", "", "config file (default $HOME/.config/kubeconsole)")
	rootCmd.PersistentFlags().StringVar(&Kubeconfig, "kubeconfig", "", "kubeconfig file (default $HOME/.kube/config)")
	rootCmd.PersistentFlags().BoolVarP(&Verbose, "verbose", "v", false, "Enable verbose")

	rootCmd.Flags().StringVarP(&options.LabelSelector, "selector", "l", "process in (console, rails-shell)", "Label selector used to filter the deployments, works the same as the -l flag for kubectl")
	rootCmd.Flags().DurationVar(&options.Timeout, "timeout", 0, "Time that the pod should live after the heartbeat has stopped. For example 15m, 24h (default 15m)")
	rootCmd.Flags().StringVar(&options.Limits, "limits", "", "The resource requirement limits for this container. For example, 'cpu=200m,memory=512Mi'. The specified limits will also be set as requests")
	rootCmd.Flags().StringVar(&options.Image, "image", "", "The image for the container to run. Replaces the image specified in the deployment")
	rootCmd.Flags().BoolVarP(&options.NoRm, "no-rm", "", false, "Do not remove pod when detaching")

	viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindPFlag("selector", rootCmd.PersistentFlags().Lookup("selector"))
	viper.BindPFlag("timeout", rootCmd.PersistentFlags().Lookup("timeout"))
}

func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if Config != "" {
		// Use config file from the flag.
		viper.SetConfigFile(Config)
	} else {
		// Search config in home directory with name ".config/kubeconsole" (without extension).
		viper.AddConfigPath(home + "/.config")
		viper.SetConfigName("kubeconsole")
		viper.SetConfigType("yaml")
	}

	if Kubeconfig == "" {
		Kubeconfig = home + "/.kube/config"
	}

	if options.Timeout < 0 {
		options.Timeout = 15 * time.Minute
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		if Verbose {
			fmt.Println("Using config file:", viper.ConfigFileUsed())
		}
	}

	MachineID, err = machineid.ID()
	if err != nil {
		panic(err)
	}

	K8sClient = k8s.NewK8s(Kubeconfig)
}
