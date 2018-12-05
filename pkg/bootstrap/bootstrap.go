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
	"context"
	"fmt"
	"time"

	"admiralty.io/multicluster-service-account/pkg/apis"
	"admiralty.io/multicluster-service-account/pkg/apis/multicluster/v1alpha1"
	"admiralty.io/multicluster-service-account/pkg/config"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var namespace = "multicluster-service-account"
var clusterRoleName = "service-account-import-controller-remote"

// TODO: allow cluster name overrides and/or get cluster names from kubeconfig instead of using context names

func Bootstrap(srcCtx string, dstCtx string) error {
	srcCfg, _, err := config.NamedConfigAndNamespace(srcCtx)
	if err != nil {
		return err
	}
	// srcClient, err := client.New(srcCfg, client.Options{})
	// if err != nil {
	// 	return err
	// }
	srcClientset, err := kubernetes.NewForConfig(srcCfg)
	if err != nil {
		return err
	}

	dstCfg, _, err := config.NamedConfigAndNamespace(dstCtx)
	if err != nil {
		return err
	}
	dstClientset, err := kubernetes.NewForConfig(dstCfg)
	if err != nil {
		return err
	}
	dstClient, err := client.New(dstCfg, client.Options{})
	if err != nil {
		return err
	}
	if err := apis.AddToScheme(scheme.Scheme); err != nil {
		return err
	}

	ns := &corev1.Namespace{}
	ns.Name = namespace
	_, err = srcClientset.CoreV1().Namespaces().Create(ns)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	cr := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName,
		},
		Rules: []rbacv1.PolicyRule{
			rbacv1.PolicyRule{
				APIGroups: []string{""},
				Resources: []string{"serviceaccounts", "secrets"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	_, err = srcClientset.RbacV1().ClusterRoles().Create(cr)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      dstCtx,
		},
	}
	_, err = srcClientset.CoreV1().ServiceAccounts(namespace).Create(sa)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: dstCtx,
		},
		RoleRef: rbacv1.RoleRef{
			Kind: "ClusterRole",
			Name: clusterRoleName,
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Namespace: sa.Namespace,
				Name:      sa.Name,
			},
		},
	}
	_, err = srcClientset.RbacV1().ClusterRoleBindings().Create(crb)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	var secretName string
	f := wait.ConditionFunc(func() (done bool, err error) {
		getSA, err := srcClientset.CoreV1().ServiceAccounts(sa.Namespace).Get(sa.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if len(getSA.Secrets) > 0 {
			secretName = getSA.Secrets[0].Name
			return true, nil
		}
		return false, nil
	})
	wait.PollImmediate(time.Second, time.Second*10, f)

	saSecret, err := srcClientset.CoreV1().Secrets(sa.Namespace).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if saSecret.Data == nil {
		return fmt.Errorf("service account token is empty")
	}

	sai := &v1alpha1.ServiceAccountImport{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      srcCtx,
		},
		Spec: v1alpha1.ServiceAccountImportSpec{
			ClusterName: srcCtx,
			Namespace:   namespace,
			Name:        dstCtx,
		},
	}
	if err := dstClient.Create(context.TODO(), sai); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// TODO: reuse code in importer
	saiSecret := &corev1.Secret{}
	saiSecret.Namespace = namespace
	saiSecret.GenerateName = srcCtx + "-token-"
	saiSecret.Data = saSecret.Data
	saiSecret.Data["server"] = []byte(srcCfg.Host)
	if err := controllerutil.SetControllerReference(sai, saiSecret, scheme.Scheme); err != nil {
		return err
	}
	saiSecret.Labels = map[string]string{
		"multicluster.admiralty.io/service-account-import.name": sai.Name,
		"multicluster.admiralty.io/remote-secret.uid":           string(saSecret.UID),
	}
	saiSecret, err = dstClientset.CoreV1().Secrets(saiSecret.Namespace).Create(saiSecret)
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	f = wait.ConditionFunc(func() (done bool, err error) {
		if err := dstClient.Get(context.TODO(), types.NamespacedName{Name: sai.Name, Namespace: sai.Namespace}, sai); err != nil {
			return false, err
		}
		if len(sai.Status.Secrets) > 0 {
			return true, nil
		}
		return false, nil
	})
	wait.PollImmediate(time.Second, time.Second*10, f)

	patch := []byte(fmt.Sprintf("{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"multicluster.admiralty.io/service-account-import.name\":\"%s\"}}}}}", sai.Name))
	_, err = dstClientset.AppsV1().Deployments(namespace).Patch("service-account-import-controller", types.StrategicMergePatchType, patch)
	if err != nil {
		return err
	}

	return nil
}
