/*
Copyright 2018 The Multicluster-Service-Account Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"admiralty.io/multicluster-service-account/pkg/bootstrap"
	"gopkg.in/alecthomas/kingpin.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	bootstrapCmd   = kingpin.Command("bootstrap", "Allow service account import controller in a cluster to import service accounts from another cluster.")
	dstCtx         = bootstrapCmd.Flag("target-context", "(default: current context) name of the kubeconfig context to use for the target cluster").String()
	dstK8sConfig   = bootstrapCmd.Flag("target-kubeconfig", "(default: KUBECONFIG environment variable or ~/.kube/config) path to kubeconfig file to use for the target cluster").ExistingFile()
	dstClusterName = bootstrapCmd.Flag("target-name", "(default: name of the kubeconfig cluster for the target context) a service account with that name will be created in the source cluster (note: use this option if, e.g., the kubeconfig cluster name isn't a valid DNS-1123 subdomain)").String()
	srcCtx         = bootstrapCmd.Flag("source-context", "(default: current context) name of the kubeconfig context to use for the source cluster").String()
	srcK8sConfig   = bootstrapCmd.Flag("source-kubeconfig", "(default: KUBECONFIG environment variable or ~/.kube/config) path to kubeconfig file to use for the source cluster").ExistingFile()
	srcClusterName = bootstrapCmd.Flag("source-name", "(default: name of the kubeconfig cluster for the source context) a service account import with that name will be created in the target cluster (note: use this option if, e.g., the kubeconfig cluster name isn't a valid DNS-1123 subdomain)").String()
)

func main() {
	kingpin.Version("0.5.1")
	kingpin.CommandLine.HelpFlag.Short('h')
	switch kingpin.Parse() {
	case "bootstrap":
		err := bootstrap.Bootstrap(*srcCtx, *srcK8sConfig, *srcClusterName, *dstCtx, *dstK8sConfig, *dstClusterName)
		if err != nil {
			kingpin.Fatalf("cannot bootstrap: %v", err)
		}
	}
}
