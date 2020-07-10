package main

import (
	"github.com/micke/kubeconsole/pkg/cmd"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	cmd.Execute()
}
