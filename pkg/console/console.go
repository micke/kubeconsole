package console

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/micke/kubeconsole/pkg/k8s"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	watchtools "k8s.io/client-go/tools/watch"
	"k8s.io/kubectl/pkg/cmd/attach"
	"k8s.io/kubectl/pkg/cmd/exec"
	generateversioned "k8s.io/kubectl/pkg/generate/versioned"
	"k8s.io/kubectl/pkg/util/interrupt"
)

// Options defines how the console should be ran
type Options struct {
	LabelSelector  string
	Timeout        time.Duration
	Command        []string
	Limits         string
	Image          string
	NoRm           bool
	DeploymentName string
	MachineID      string
}

var (
	defaultAttachTimeout = 30 * time.Second
	errInterrupted       = errors.New("interrupted")
)

// Start the console
func Start(k8s *k8s.K8s, options Options) {
	deployments := k8s.Deployments(options.LabelSelector)

	if len(deployments) == 0 {
		fmt.Fprintf(os.Stderr, "No mathing deployments found. label-selector is currently: %s\n", options.LabelSelector)
		os.Exit(1)
	}

	deployment := selectDeployment(deployments, options.DeploymentName)

	user, err := user.Current()
	if err != nil {
		panic(err)
	}

	podsClient := k8s.Clientset.CoreV1().Pods(deployment.Namespace)
	pod := &apiv1.Pod{
		Spec:       deployment.Spec.Template.Spec,
		ObjectMeta: deployment.Spec.Template.ObjectMeta,
	}

	if pod.Labels == nil {
		pod.Labels = map[string]string{}
	}
	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}
	pod.Labels["kubeconsole.garbagecollect"] = "true"
	pod.Labels["kubeconsole.creator.machineid"] = options.MachineID
	pod.Annotations["kubeconsole.creator.username"] = user.Username
	pod.Annotations["kubeconsole.creator.name"] = user.Name
	pod.Annotations["kubeconsole.heartbeat"] = time.Now().Format(time.RFC3339)
	pod.Annotations["kubeconsole.timeout"] = strconv.Itoa(int(options.Timeout.Minutes()))

	pod.Spec.RestartPolicy = apiv1.RestartPolicyNever
	pod.Spec.Containers[0].TTY = true
	pod.Spec.Containers[0].Stdin = true

	// Set command if one was provided
	if len(options.Command) > 0 {
		pod.Spec.Containers[0].Command = options.Command
	}

	// Set default GenerateName if it's not already set
	if pod.GenerateName == "" {
		pod.GenerateName = "kubeconsole-"
	}

	// Set limits and requests
	if options.Limits != "" {
		params := map[string]string{}
		params["limits"] = options.Limits
		params["requests"] = options.Limits
		resourceRequirements, err := generateversioned.HandleResourceRequirementsV1(params)
		if err != nil {
			panic(err)
		}
		pod.Spec.Containers[0].Resources = resourceRequirements
	}

	// Set image if one was specified
	if options.Image != "" {
		pod.Spec.Containers[0].Image = options.Image
	}

	// Find existing pod if one exists
	attachablePod := findRunningPod(pod, podsClient)

	// If no running pod is found we will create one
	if attachablePod == nil {
		attachablePod, err = podsClient.Create(context.TODO(), pod, metav1.CreateOptions{})
		if err != nil {
			panic(err)
		}
		fmt.Printf("Created pod %s/%s\n", attachablePod.Namespace, attachablePod.Name)
	}

	if !options.NoRm {
		defer deletePod(attachablePod, podsClient)
	}
	go watchPodEvents(attachablePod, k8s.Clientset)
	scheduleHeartbeat(attachablePod, podsClient)

	attachOpts := &attach.AttachOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: genericclioptions.IOStreams{
				In:     os.Stdin,
				Out:    os.Stdout,
				ErrOut: os.Stderr,
			},
			Stdin: true,
			TTY:   true,
			Quiet: true,
		},
		GetPodTimeout: defaultAttachTimeout,
		Attach:        &attach.DefaultRemoteAttach{},
		Config:        k8s.RestConfig,
		AttachFunc:    attach.DefaultAttachFunc,
	}

	err = handleAttachPod(podsClient, attachablePod, attachOpts)
	if err != nil && err != errInterrupted {
		panic(err)
	}
}

// List lists all running console pods
func List(k8s *k8s.K8s, environments []string, everyone bool, machineID string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ENVIRONMENT\tNAME\tNAMESPACE\tCREATOR\tAGE\tIMAGE\tLABELS")

	selectors := map[string]string{
		"kubeconsole.garbagecollect": "true",
	}

	if !everyone {
		selectors["kubeconsole.creator.machineid"] = machineID
	}

	environmentWriters := make(map[string]*bufio.Writer, 0)
	var wg sync.WaitGroup

	for _, environment := range environments {
		wg.Add(1)
		environmentWriters[environment] = bufio.NewWriter(w)

		go func(environment string) {
			k8s.SelectContext(environment)

			podsClient := k8s.Clientset.CoreV1().Pods("")
			pods, err := podsClient.List(
				context.TODO(),
				metav1.ListOptions{LabelSelector: fields.SelectorFromSet(selectors).String()},
			)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching pods for %s: %s\n", environment, err)
			}

			for _, p := range pods.Items {
				fmt.Fprintf(
					environmentWriters[environment],
					"%s\t%s\t%s\t%s\t%s\t%v\t%v\n",
					environment,
					p.Name,
					p.Namespace,
					p.Annotations["kubeconsole.creator.name"],
					formatAge(p.CreationTimestamp.Time),
					p.Spec.Containers[0].Image,
					formatLabels(p.Labels),
				)
			}

			wg.Done()
		}(environment)
	}

	wg.Wait()

	for _, writer := range environmentWriters {
		writer.Flush()
	}

	w.Flush()
}

func formatAge(datetime time.Time) string {
	duration := time.Now().Sub(datetime)

	if duration.Hours() > 24 {
		return fmt.Sprintf("%.0fd", math.RoundToEven(duration.Hours()/24))
	} else if duration.Minutes() > 60 {
		return fmt.Sprintf("%.0fh", math.RoundToEven(duration.Hours()))
	} else {
		return fmt.Sprintf("%.0fm", math.RoundToEven(duration.Minutes()))
	}
}

func formatLabels(labels map[string]string) string {
	var formattedLabels []string

	for name, value := range labels {
		if strings.HasPrefix(name, "kubeconsole.") {
			continue
		}

		formattedLabels = append(formattedLabels, fmt.Sprintf("%s=%s", name, value))
	}

	return strings.Join(formattedLabels, " ")
}

func selectDeployment(allDeployments []appsv1.Deployment, deploymentName string) *appsv1.Deployment {
	var deployments []appsv1.Deployment

	if deploymentName != "" {
		// If deploymentName is specified then we will filter for deployments matching it
		for _, d := range allDeployments {
			if strings.HasPrefix(d.Name, deploymentName) {
				deployments = append(deployments, d)
			}
		}

		// If exactly one deployment matches deploymentName then that's the deployment we want to run
		if len(deployments) == 1 {
			return &deployments[0]
		}
	}

	// If no deployments matched deploymentName then we let the user pick among all of them
	if len(deployments) == 0 {
		deployments = allDeployments
	}

	deploymentNames := make([]string, len(deployments))
	for i, d := range deployments {
		deploymentNames[i] = d.Name
	}

	selectedDeployment := 0
	prompt := &survey.Select{
		Message: "Choose a deployment:",
		Options: deploymentNames,
	}
	err := survey.AskOne(prompt, &selectedDeployment)
	if err == terminal.InterruptErr {
		fmt.Println("Cancelled")
		os.Exit(0)
	} else if err != nil {
		panic(err)
	}

	return &deployments[selectedDeployment]
}

func deletePod(pod *apiv1.Pod, podsClient v1.PodInterface) {
	err := podsClient.Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})
	if err == nil {
		fmt.Printf("\nDeleted pod %s/%s\n", pod.Namespace, pod.Name)
	} else {
		fmt.Printf("Failed to delete pod %s/%s: %s\n", pod.Namespace, pod.Name, err)
	}
}

func waitForPod(podsClient v1.PodInterface, pod *apiv1.Pod, exitCondition watchtools.ConditionFunc) (*apiv1.Pod, error) {
	// TODO: expose the timeout
	ctx, cancel := watchtools.ContextWithOptionalTimeout(context.Background(), 0)
	defer cancel()

	preconditionFunc := func(store cache.Store) (bool, error) {
		_, exists, err := store.Get(&metav1.ObjectMeta{Namespace: pod.Namespace, Name: pod.Name})
		if err != nil {
			return true, err
		}
		if !exists {
			// We need to make sure we see the object in the cache before we start waiting for events
			// or we would be waiting for the timeout if such object didn't exist.
			// (e.g. it was deleted before we started informers so they wouldn't even see the delete event)
			return true, apierrors.NewNotFound(apiv1.Resource("pods"), pod.Name)
		}

		return false, nil
	}

	fieldSelector := fields.OneTermEqualSelector("metadata.name", pod.Name).String()
	lw := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			options.FieldSelector = fieldSelector
			return podsClient.List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			options.FieldSelector = fieldSelector
			return podsClient.Watch(context.TODO(), options)
		},
	}

	var interrupted bool
	var result *apiv1.Pod
	intr := interrupt.New(func(s os.Signal) { interrupted = true }, cancel)
	err := intr.Run(func() error {
		ev, err := watchtools.UntilWithSync(ctx, lw, &apiv1.Pod{}, preconditionFunc, func(ev watch.Event) (bool, error) {
			return exitCondition(ev)
		})
		if ev != nil {
			result = ev.Object.(*apiv1.Pod)
		}
		return err
	})

	if interrupted {
		return result, errInterrupted
	}

	return result, err
}

func findRunningPod(pod *apiv1.Pod, podsClient v1.PodInterface) *apiv1.Pod {
	pods, err := podsClient.List(
		context.TODO(),
		metav1.ListOptions{
			LabelSelector: fields.SelectorFromSet(pod.Labels).String(),
			FieldSelector: "status.phase=Running",
		},
	)

	if err != nil {
		fmt.Println("Error finding already running consoles. Defaulting to creating new one")
		return nil
	}

	if len(pods.Items) == 0 {
		return nil
	}

	selectedPod := 0
	options := make([]string, len(pods.Items)+1)
	options[0] = "Create a new console pod"

	for i, pod := range pods.Items {
		options[i+1] = fmt.Sprintf("%s: Creeated %s", pod.Spec.Containers[0].Command, pod.CreationTimestamp)
	}

	prompt := &survey.Select{
		Message: "Existing console pod found, would you like to create a new one or attach to an existing one?",
		Options: options,
	}
	err = survey.AskOne(prompt, &selectedPod)

	if err == terminal.InterruptErr {
		fmt.Println("Cancelled")
		os.Exit(0)
	} else if err != nil {
		panic(err)
	}

	if selectedPod == 0 {
		return nil
	}

	return &pods.Items[selectedPod-1]
}

func handleAttachPod(podsClient v1.PodInterface, pod *apiv1.Pod, attachOpts *attach.AttachOptions) error {
	pod, err := waitForPod(podsClient, pod, podRunningAndReady)
	if err != nil && err != ErrPodCompleted {
		return err
	}

	if pod.Status.Phase == apiv1.PodSucceeded || pod.Status.Phase == apiv1.PodFailed {
		return fmt.Errorf("Pod failed or ran to completion")
	}

	attachOpts.Pod = pod
	attachOpts.PodName = pod.Name
	attachOpts.Namespace = pod.Namespace

	fmt.Print("Attaching...\n")

	if err := attachOpts.Run(); err != nil {
		fmt.Fprintf(attachOpts.ErrOut, "Error attaching, falling back to logs: %v\n", err)
	}
	return nil
}

// ErrPodCompleted is returned by PodRunning or PodContainerRunning to indicate that
// the pod has already reached completed state.
var ErrPodCompleted = fmt.Errorf("pod ran to completion")

// podRunningAndReady returns true if the pod is running and ready, false if the pod has not
// yet reached those states, returns ErrPodCompleted if the pod has run to completion, or
// an error in any other case.
func podRunningAndReady(event watch.Event) (bool, error) {
	switch event.Type {
	case watch.Deleted:
		return false, apierrors.NewNotFound(schema.GroupResource{Resource: "pods"}, "")
	}
	switch t := event.Object.(type) {
	case *apiv1.Pod:
		switch t.Status.Phase {
		case apiv1.PodFailed, apiv1.PodSucceeded:
			return false, ErrPodCompleted
		case apiv1.PodRunning:
			conditions := t.Status.Conditions
			if conditions == nil {
				return false, nil
			}
			for i := range conditions {
				if conditions[i].Type == apiv1.PodReady &&
					conditions[i].Status == apiv1.ConditionTrue {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func watchPodEvents(pod *apiv1.Pod, clientset *kubernetes.Clientset) {
	fieldSelector := fields.OneTermEqualSelector("involvedObject.uid", string(pod.UID))
	watchlist := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "events", pod.Namespace, fieldSelector)
	_, controller := cache.NewInformer(
		watchlist,
		&apiv1.Event{},
		time.Second*0,
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				event := obj.(*apiv1.Event)
				fmt.Printf("%s\n", event.Message)
			},
		},
	)
	stop := make(chan struct{})
	controller.Run(stop)
}

func heartbeat(pod *apiv1.Pod, podsClient v1.PodInterface) error {
	patch := fmt.Sprintf(
		`{"metadata":{"annotations":{"kubeconsole.heartbeat":"%s"}}}`,
		time.Now().Format(time.RFC3339),
	)

	_, err := podsClient.Patch(context.TODO(), pod.Name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		fmt.Printf("Error updating heartbeat on pod: %+v\n", err)
		return err
	}

	return nil
}

func scheduleHeartbeat(pod *apiv1.Pod, podsClient v1.PodInterface) {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for t := range ticker.C {
			_ = t
			heartbeat(pod, podsClient)
		}
	}()
}
