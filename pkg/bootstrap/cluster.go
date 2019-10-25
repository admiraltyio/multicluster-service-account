/*
Copyright 2019 The Multicluster-Service-Account Authors.

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

package bootstrap

import (
	"fmt"

	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cluster struct {
	name      string
	client    client.Client
	clientset *kubernetes.Clientset
	host      string
}

func newCluster(name, kubeconfigPath, context string) (cluster, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = kubeconfigPath                              // if empty, env var or default path will be used instead
	overrides := &clientcmd.ConfigOverrides{CurrentContext: context} // if context is empty, kubeconfig's current context will be used instead
	cfgLoader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides)
	cfgRaw, err := cfgLoader.RawConfig()
	if err != nil {
		return cluster{}, err // TODO allow running bootstrap with in-cluster config
	}

	if name == "" {
		if context == "" {
			context = cfgRaw.CurrentContext
		}
		name = cfgRaw.Contexts[context].Cluster
	}
	if msgs := validation.IsDNS1123Subdomain(name); len(msgs) != 0 {
		return cluster{}, fmt.Errorf("invalid cluster name \"%s\", must be a DNS-1123 subdomain (rename it in the kubeconfig or use the --target-name/--source-name option to override it): %v", name, msgs)
	}

	cfg, err := cfgLoader.ClientConfig()
	if err != nil {
		return cluster{}, err
	}
	client, err := client.New(cfg, client.Options{})
	if err != nil {
		return cluster{}, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return cluster{}, err
	}

	return cluster{
		name:      name,
		client:    client,
		clientset: clientset,
		host:      cfg.Host,
	}, nil
}
