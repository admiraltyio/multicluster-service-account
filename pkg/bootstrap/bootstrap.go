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

package bootstrap

import (
	"admiralty.io/multicluster-service-account/pkg/apis"
	"fmt"
	"k8s.io/client-go/kubernetes/scheme"
)

var namespace = "multicluster-service-account"
var deployName = "service-account-import-controller"
var clusterRoleName = "service-account-import-controller-remote"

func Bootstrap(srcCtx, srcKubeconfig, srcClusterName, dstCtx, dstKubeconfig, dstClusterName string) error {
	src, err := newCluster(srcClusterName, srcKubeconfig, srcCtx)
	if err != nil {
		return fmt.Errorf("cannot load source cluster: %v", err)
	}
	dst, err := newCluster(dstClusterName, dstKubeconfig, dstCtx)
	if err != nil {
		return fmt.Errorf("cannot load target cluster: %v", err)
	}
	return bootstrapClusters(src, dst)
}

func bootstrapClusters(source, target cluster) error {
	srcCluster := sourceCluster{source}
	dstCluster := targetCluster{target}

	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		return err
	}

	// The source cluster may not have have multicluster-service-account installed,
	// but it needs a service account that can read other service accounts and their token secrets.
	// We create that service account in the multicluster-service-account namespace,
	// and create that namespace if it doesn't exist.
	err := srcCluster.createNamespace()
	if err != nil {
		return err
	}
	err = srcCluster.createClusterRole()
	if err != nil {
		return err
	}
	err = srcCluster.createServiceAccount(dstCluster.name)
	if err != nil {
		return err
	}
	err = srcCluster.createClusterRoleBinding(dstCluster.name)
	if err != nil {
		return err
	}
	secretName, err := srcCluster.waitForServiceAccountToken(dstCluster.name)
	if err != nil {
		return err
	}
	saSecret, err := srcCluster.getServiceAccountToken(secretName)
	if err != nil {
		return err
	}

	sai, err := dstCluster.createServiceAccountImport(srcCluster.name)
	if err != nil {
		return err
	}
	err = dstCluster.createServiceAccountImportToken(sai, saSecret, srcCluster.host)
	if err != nil {
		return err
	}
	err = dstCluster.waitForServiceAccountImportTokenAdoption(srcCluster.name)
	if err != nil {
		return err
	}
	err = dstCluster.annotateServiceAccountController(srcCluster.name)
	if err != nil {
		return err
	}

	return nil
}
