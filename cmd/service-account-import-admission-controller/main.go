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
	"log"

	"admiralty.io/multicluster-service-account/pkg/apis"
	"admiralty.io/multicluster-service-account/pkg/automount"
	"admiralty.io/multicluster-service-account/pkg/config"
	"k8s.io/sample-controller/pkg/signals"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func main() {
	cfg, ns, err := config.ConfigAndNamespace()
	if err != nil {
		log.Fatalf("cannot get config and namespace: %v", err)
	}

	m, err := manager.New(cfg, manager.Options{})
	if err != nil {
		log.Fatalf("cannot create manager: %v", err)
	}

	if err := apis.AddToScheme(m.GetScheme()); err != nil {
		log.Fatalf("cannot add APIs to scheme: %v", err)
	}

	_, err = automount.NewServer(m, ns)
	if err != nil {
		log.Fatalf("cannot create automount server: %v", err)
	}

	if err := m.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalf("while or after starting manager: %v", err)
	}
}
