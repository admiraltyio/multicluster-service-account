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
	"admiralty.io/multicluster-service-account/pkg/config"
	"admiralty.io/multicluster-service-account/pkg/importer"
	"github.com/ghodss/yaml"
	"k8s.io/client-go/kubernetes"
)

func Export(kubeconfig, context, namespace, name, exportName string) ([]byte, error) {
	cfg, ns, err := config.ConfigAndNamespaceForKubeconfigAndContext(kubeconfig, context)
	if err != nil {
		return nil, err
	}
	if namespace == "" {
		namespace = ns
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	secretName, err := waitForServiceAccountToken(clientset, namespace, name)
	if err != nil {
		return nil, err
	}
	saSecret, err := getServiceAccountToken(clientset, namespace, secretName)
	if err != nil {
		return nil, err
	}
	if exportName == "" {
		exportName = saSecret.Name
	}
	exportSecret := importer.ExportServiceAccountSecret(saSecret, cfg.Host, exportName)
	exportSecret.Name = exportName
	// HACK fill un TypeMeta TODO use apimachinery encoder
	exportSecret.APIVersion = "v1"
	exportSecret.Kind = "Secret"
	out, err := yaml.Marshal(exportSecret)
	if err != nil {
		return nil, err
	}
	return out, nil
}
