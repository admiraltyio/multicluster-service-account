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
	bootstrapCmd = kingpin.Command("bootstrap", "Allow service account import controller in a cluster to import service accounts from another cluster.")
	dstCtx       = bootstrapCmd.Arg("target", "(default: current context) kubeconfig context corresponding to the cluster INTO which you want to import service accounts").Required().String()
	srcCtx       = bootstrapCmd.Arg("source", "kubeconfig context corresponding to the cluster FROM which you want to import service accounts").Required().String()
)

func main() {
	kingpin.Version("0.4.1")
	kingpin.CommandLine.HelpFlag.Short('h')
	switch kingpin.Parse() {
	case "bootstrap":
		err := bootstrap.Bootstrap(*srcCtx, *dstCtx)
		if err != nil {
			kingpin.Fatalf("cannot bootstrap: %v", err)
		}
	}
}
