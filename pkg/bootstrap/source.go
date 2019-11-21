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
	"time"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

type sourceCluster struct {
	cluster
}

func (c sourceCluster) createNamespace() error {
	ns := &corev1.Namespace{}
	ns.Name = namespace
	_, err := c.clientset.CoreV1().Namespaces().Create(ns)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		fmt.Printf("namespace \"%s\" already exists in source cluster \"%s\"\n", namespace, c.name)
	} else {
		fmt.Printf("created namespace \"%s\" in source cluster \"%s\"\n", namespace, c.name)
	}
	return nil
}

func (c sourceCluster) createClusterRole() error {
	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts", "secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	_, err := c.clientset.RbacV1().ClusterRoles().Create(cr)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		fmt.Printf("cluster role \"%s\" already exists in source cluster \"%s\"\n", clusterRoleName, c.name)
	} else {
		fmt.Printf("created cluster role \"%s\" in source cluster \"%s\"\n", clusterRoleName, c.name)
	}
	return nil
}

func (c sourceCluster) createServiceAccount(targetClusterName string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      targetClusterName,
		},
	}
	_, err := c.clientset.CoreV1().ServiceAccounts(namespace).Create(sa)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		fmt.Printf("service account \"%s\" already exists in namespace \"%s\" in source cluster \"%s\"\n", sa.Name, namespace, c.name)
	} else {
		fmt.Printf("created service account \"%s\" in namespace \"%s\" in source cluster \"%s\"\n", sa.Name, namespace, c.name)
	}
	return nil
}

func (c sourceCluster) createClusterRoleBinding(targetClusterName string) error {
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: targetClusterName,
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: clusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: namespace,
				Name:      targetClusterName,
			},
		},
	}
	_, err := c.clientset.RbacV1().ClusterRoleBindings().Create(crb)
	if err != nil {
		if !errors.IsAlreadyExists(err) {
			return err
		}
		fmt.Printf("cluster role binding \"%s\" already exists in source cluster \"%s\"\n", crb.Name, c.name)
	} else {
		fmt.Printf("created cluster role binding \"%s\" in source cluster \"%s\"\n", crb.Name, c.name)
	}
	return nil
}

func (c sourceCluster) waitForServiceAccountToken(targetClusterName string) (secretName string, err error) {
	fmt.Printf("waiting until service account \"%s\" in namespace \"%s\" in source cluster \"%s\" has a token...\n", targetClusterName, namespace, c.name)
	secretName, err = waitForServiceAccountToken(c.clientset, namespace, targetClusterName)
	if err != nil {
		return "", fmt.Errorf("in source cluster \"%s\": %v", c.name, err)
	}
	return secretName, nil
}

func waitForServiceAccountToken(clientset *kubernetes.Clientset, namespace, name string) (secretName string, err error) {
	f := wait.ConditionFunc(func() (done bool, err error) {
		getSA, err := clientset.CoreV1().ServiceAccounts(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("cannot get service account \"%s\" in namespace \"%s\": %v", name, namespace, err)
		}
		if len(getSA.Secrets) > 0 {
			secretName = getSA.Secrets[0].Name
			return true, nil
		}
		return false, nil
	})
	if err := wait.PollImmediate(time.Second, time.Minute, f); err != nil {
		return "", fmt.Errorf("timeout: %v", err)
	}
	return secretName, nil
}

func (c sourceCluster) getServiceAccountToken(secretName string) (*corev1.Secret, error) {
	saSecret, err := getServiceAccountToken(c.clientset, namespace, secretName)
	if err != nil {
		return nil, fmt.Errorf("in source cluster \"%s\": %v", c.name, err)
	}
	return saSecret, nil
}

func getServiceAccountToken(clientset *kubernetes.Clientset, namespace, name string) (*corev1.Secret, error) {
	saSecret, err := clientset.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot get secret \"%s\" in namespace \"%s\": %v", name, namespace, err)
	}
	if saSecret.Data == nil {
		return nil, fmt.Errorf("secret \"%s\" in namespace \"%s\" is empty", name, namespace)
	}
	return saSecret, nil
}
