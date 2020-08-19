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

Comming soon


# How to use

kubeconsole currently expects your environments to be separated into different
kubectl contexts, so to run a console in your production cluster you execute `kubeconosole production`.
