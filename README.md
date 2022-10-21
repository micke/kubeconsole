![Logo](https://raw.githubusercontent.com/micke/kubeconsole/master/assets/logosmall.png)

We all want to live in a perfect world where our applications have awesome UI's
where you can configure and control everything.

Unfortunately that's not the world we live in and some times you need to get
down and dirty in a REPL or a shell.

This is easy to do when running the application on a VM or a dedicated server,
you just SSH into the machine and run the command.  
But how do you accomplish this in kubernetes? A common solution is to run a
deployment with one replica that just runs the `sleep infinitely` command
waiting for someone to `kubectl exec` into it to run the REPL or shell.

There is some big issues with this solution though, for example what happens if
the deployment is updated? or maybe several developers are using the same pod
and exceeds the memory limit?

`kubeconsole` solves this by letting you define how the console should run by
configuring a regular deployment, but without any running replicas.
And when a developer needs a REPL he/she will just run `kubeconsole` and
kubeconsole will find all the kubeconsole deployments running in the cluster and
start a single pod for that instance of the console. Killing the pod when the
command exits (or when a configurable timout is exceeded if running the
companion controller in the cluster).

# Installation

## MacOS

`brew install micke/kubeconsole/kubeconsole`

## Linux

[Download the latest pre-built binary](https://github.com/micke/kubeconsole/releases) for your architecture and place it in your `PATH`.


# How to use

kubeconsole currently expects your environments to be separated into different
kubectl contexts, so to run a console in your production cluster you execute `kubeconosole production`.

## More info see `kubeconsole -h`

```
kubeconsole is used to create a temporary pod based on a deployments specification

Usage:
  kubeconsole [environment] [flags]
  kubeconsole [command]

Examples:
# Select a deployment in the production environment
kubeconsole production
# Run a custom command instead of the command specified in the deployment
kubeconsole production -- /bin/bash

Available Commands:
  completion  Generate completion script
  help        Help about any command
  ls          Lists all the currently running console pods

Flags:
  -c, --config string       config file (default $HOME/.config/kubeconsole)
  -h, --help                help for kubeconsole
      --image string        The image for the container to run. Replaces the image specified in the deployment
      --kubeconfig string   kubeconfig file (default $HOME/.kube/config)
      --limits string       The resource requirement limits for this container. For example, 'cpu=200m,memory=512Mi'. The specified limits will also be set as requests
      --no-rm               Do not remove pod when detaching
  -l, --selector string     Label selector used to filter the deployments, works the same as the -l flag for kubectl (default "process in (console, rails-shell)")
      --timeout duration    Time that the pod should live after the heartbeat has stopped. For example 15m, 24h (default 15m0s)
  -v, --verbose             Enable verbose

Use "kubeconsole [command] --help" for more information about a command.
```
