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
	"context"
	"fmt"
	"time"

	"admiralty.io/multicluster-service-account/pkg/apis/multicluster/v1alpha1"
	"admiralty.io/multicluster-service-account/pkg/importer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

type targetCluster struct {
	cluster
}

func (c targetCluster) createServiceAccountImport(sourceClusterName string) (*v1alpha1.ServiceAccountImport, error) {
	sai := &v1alpha1.ServiceAccountImport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      sourceClusterName,
		},
		Spec: v1alpha1.ServiceAccountImportSpec{
			ClusterName: sourceClusterName,
			Namespace:   namespace,
			Name:        c.name,
		},
	}
	if err := c.client.Create(context.TODO(), sai); err != nil {
		if !errors.IsAlreadyExists(err) {
			return nil, err
		}
		fmt.Printf("service account import \"%s\" already exists in namespace \"%s\" in target cluster \"%s\"\n", sai.Name, sai.Namespace, c.name)
		// in this case, the server doesn't return the state of sai, therefore it's missing a uid,
		// and the controller reference created below on the secret would be invalid if we do not get it
		if err := c.client.Get(context.TODO(), types.NamespacedName{Name: sai.Name, Namespace: sai.Namespace}, sai); err != nil {
			return nil, fmt.Errorf("cannot get service account import \"%s\" in namespace \"%s\" in target cluster \"%s\": %v", sai.Name, sai.Namespace, c.name, err)
		}
	} else {
		fmt.Printf("created service account import \"%s\" in namespace \"%s\" in target cluster \"%s\"\n", sai.Name, sai.Namespace, c.name)
	}
	return sai, nil
}

func (c targetCluster) createServiceAccountImportToken(sai *v1alpha1.ServiceAccountImport, sourceServiceAccountToken *corev1.Secret, sourceClusterCfg *rest.Config) error {
	saiSecret := importer.MakeServiceAccountImportSecret(sai, sourceServiceAccountToken, sourceClusterCfg, scheme.Scheme)
	saiSecret, err := c.clientset.CoreV1().Secrets(saiSecret.Namespace).Create(saiSecret)
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("cannot create secret \"%s\" in namespace \"%s\" in target cluster \"%s\": %v", saiSecret.GenerateName, saiSecret.Namespace, c.name, err)
	}
	fmt.Printf("created secret \"%s\" in namespace \"%s\" in target cluster \"%s\"\n", saiSecret.GenerateName, saiSecret.Namespace, c.name)
	return nil
}

func (c targetCluster) waitForServiceAccountImportTokenAdoption(sourceClusterName string) error {
	fmt.Printf("waiting until service account import \"%s\" in namespace \"%s\" in target cluster \"%s\" adopts token...\n", sourceClusterName, namespace, c.name)
	sai := &v1alpha1.ServiceAccountImport{}
	f := wait.ConditionFunc(func() (done bool, err error) {
		if err := c.client.Get(context.TODO(), types.NamespacedName{Name: sourceClusterName, Namespace: namespace}, sai); err != nil {
			return false, fmt.Errorf("cannot get service account import \"%s\" in namespace \"%s\" in target cluster \"%s\": %v", sourceClusterName, namespace, c.name, err)
		}
		if len(sai.Status.Secrets) > 0 {
			return true, nil
		}
		return false, nil
	})
	if err := wait.PollImmediate(time.Second, time.Minute, f); err != nil {
		return fmt.Errorf("timeout: %v", err)
	}
	return nil
}

func (c targetCluster) annotateServiceAccountController(sourceClusterName string) error {
	deploy, err := c.clientset.AppsV1().Deployments(namespace).Get(deployName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("cannot get deployment \"%s\" in namespace \"%s\" in target cluster \"%s\": %v", deployName, namespace, c.name, err)
	}
	imports, _ := deploy.Spec.Template.Annotations[importer.AnnotationKeyServiceAccountImportName]
	if imports != "" {
		imports += ","
	}
	imports += sourceClusterName
	deployCopy := deploy.DeepCopy()
	if deployCopy.Spec.Template.Annotations == nil {
		deployCopy.Spec.Template.Annotations = map[string]string{importer.AnnotationKeyServiceAccountImportName: imports}
	} else {
		deployCopy.Spec.Template.Annotations[importer.AnnotationKeyServiceAccountImportName] = imports
	}
	_, err = c.clientset.AppsV1().Deployments(namespace).Update(deployCopy)
	if err != nil {
		return fmt.Errorf("cannot annotate service account import controller in target cluster \"%s\": %v", c.name, err)
	}
	fmt.Printf("annotated service account import controller in target cluster \"%s\"\n", c.name)
	return nil
}
