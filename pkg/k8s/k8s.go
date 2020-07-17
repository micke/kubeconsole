package k8s

import (
	"context"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/kubectl/pkg/scheme"
)

// K8s is a struct of the commonly used kubernetes client
type K8s struct {
	Config     api.Config
	RestConfig *rest.Config
	Clientset  *kubernetes.Clientset
	Contexts   map[string]*api.Context
}

// NewK8s initializes a K8s
func NewK8s(kubeconfig string) *K8s {
	clientConfig := clientConfig(kubeconfig)
	config, err := clientConfig.RawConfig()
	if err != nil {
		panic(err)
	}

	return &K8s{
		Config:   config,
		Contexts: config.Contexts,
	}
}

// ContextNames returns the contexts available in a kubeconfig
func (k8s *K8s) ContextNames() []string {
	contextNames := make([]string, 0, len(k8s.Contexts))

	for key := range k8s.Contexts {
		contextNames = append(contextNames, key)
	}

	return contextNames
}

// ContextNamesWithPrefix returns the context names that begins with the passed prefix
func (k8s *K8s) ContextNamesWithPrefix(prefix string) []string {
	matchingContexts := []string{}

	for _, context := range k8s.ContextNames() {
		if strings.HasPrefix(context, prefix) {
			matchingContexts = append(matchingContexts, context)
		}
	}

	return matchingContexts
}

// Deployments returns a list of deployments
func (k8s *K8s) Deployments(labelSelector string) []appsv1.Deployment {
	deploymentsClient := k8s.Clientset.AppsV1().Deployments("")

	list, err := deploymentsClient.List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		panic(err)
	}
	deployments := list.Items

	return deployments
}

// DeploymentNamesWithPrefix returns the context names that begins with the passed prefix
func (k8s *K8s) DeploymentNamesWithPrefix(prefix string, labelSelector string) []string {
	matchingDeploys := []string{}

	for _, deploy := range k8s.Deployments(labelSelector) {
		if strings.HasPrefix(deploy.Name, prefix) {
			matchingDeploys = append(matchingDeploys, deploy.Name)
		}
	}

	return matchingDeploys
}

// SelectContext selects a context to connect the clientset to
func (k8s *K8s) SelectContext(context string) {
	override := &clientcmd.ConfigOverrides{CurrentContext: context}
	clientConfig := clientcmd.NewNonInteractiveClientConfig(
		k8s.Config,
		override.CurrentContext,
		override,
		&clientcmd.ClientConfigLoadingRules{},
	)

	config, err := clientConfig.ClientConfig()
	if err != nil {
		panic(err)
	}

	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	config.APIPath = "/api"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	k8s.RestConfig = config
	k8s.Clientset = clientset
}

func clientConfig(kubeconfig string) clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{Precedence: []string{kubeconfig}},
		&clientcmd.ConfigOverrides{},
	)
}
