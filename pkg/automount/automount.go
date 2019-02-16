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

package automount // import "admiralty.io/multicluster-service-account/pkg/automount"

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"admiralty.io/multicluster-service-account/pkg/apis/multicluster/v1alpha1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
	atypes "sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

var (
	AnnotationKeyServiceAccountImportName = "multicluster.admiralty.io/service-account-import.name"
	webhookName                           = "service-account-import-admission-controller.multicluster.admiralty.io"
)

// +kubebuilder:webhook:port=9876,cert-dir=/tmp/cert
// +kubebuilder:webhook:service=multicluster-service-account-webhook:service-account-import-admission-controller
// +kubebuilder:webhook:selector=app:service-account-import-admission-controller
// +kubebuilder:webhook:secret=multicluster-service-account-webhook:service-account-import-admission-controller-cert
// +kubebuilder:webhook:mutating-webhook-config-name=service-account-import-admission-controller

func NewServer(mgr manager.Manager, namespace string) (*webhook.Server, error) {
	w, err := NewWebhook(mgr)
	if err != nil {
		return nil, fmt.Errorf("cannot build webhook: %v", err)
	}

	deployName := os.Getenv("DEPLOYMENT_NAME")

	s, err := webhook.NewServer(deployName, mgr, webhook.ServerOptions{
		Port:    9876, // TODO debug why cannot default to 443
		CertDir: "/tmp/cert",
		BootstrapOptions: &webhook.BootstrapOptions{
			MutatingWebhookConfigName: deployName,
			Secret: &types.NamespacedName{
				Namespace: namespace,
				Name:      deployName + "-cert",
			},
			Service: &webhook.Service{
				Namespace: namespace,
				Name:      deployName,
				// Selectors should select the pods that runs this webhook server.
				Selectors: map[string]string{
					"app": deployName,
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("cannot create server: %v", err)
	}

	if err := s.Register(w); err != nil {
		return nil, fmt.Errorf("cannot register webhook with server: %v", err)
	}

	return s, nil
}

// +kubebuilder:webhook:name=service-account-import-admission-controller.multicluster.admiralty.io
// +kubebuilder:webhook:type=mutating
// +kubebuilder:webhook:path=/mutate-pods
// +kubebuilder:webhook:verbs=create
// +kubebuilder:webhook:groups=,versions=v1,resources=pods
// +kubebuilder:webhook:failure-policy=fail

// TODO: annotation for namespace selector

// https://kubernetes.slack.com/archives/CAR30FCJZ/p1547254570666900
func NewWebhook(mgr manager.Manager) (*admission.Webhook, error) {
	return builder.NewWebhookBuilder().
		Name(webhookName).
		Mutating().
		Path("/mutate-pods").
		Operations(admissionregistrationv1beta1.Create).
		ForType(&corev1.Pod{}).
		FailurePolicy(admissionregistrationv1beta1.Fail).
		NamespaceSelector(&metav1.LabelSelector{MatchLabels: map[string]string{"multicluster-service-account": "enabled"}}).
		WithManager(mgr).
		Handlers(&Handler{}).
		Build()
}

// Handler handles pod admission requests, mutating pods that request service account imports.
// It is implemented by the service-account-import-admission-controller command, via controller-runtime.
// If a pod is annotated with the "multicluster.admiralty.io/service-account-import.name" key,
// where the value is a comma-separated list of service account import names, for each
// service account import, a volume is added to the pod, sourced from the first secret
// listed by the service account import, and mounted in each of the pod's containers under
// /var/run/secrets/admiralty.io/serviceaccountimports/%s, where %s is the service account import name.
type Handler struct {
	decoder atypes.Decoder
	client  client.Client
}

func (h *Handler) Handle(ctx context.Context, req atypes.Request) atypes.Response {
	pod := &corev1.Pod{}
	err := h.decoder.Decode(req, pod)
	if err != nil {
		err := fmt.Errorf("cannot decode admission request for object %s in namespace %s: %v",
			req.AdmissionRequest.Name, req.AdmissionRequest.Namespace, err)
		log.Println(err)
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}

	copy := pod.DeepCopy()
	err = h.mutatePodsFn(ctx, req, copy)
	if err != nil {
		err := fmt.Errorf("cannot handle admission request for pod %s in namespace %s: %v",
			getName(pod, req.AdmissionRequest), getNamespace(pod, req.AdmissionRequest), err)
		log.Println(err)
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}

	return admission.PatchResponse(pod, copy)
}

func getName(pod *corev1.Pod, req *admissionv1beta1.AdmissionRequest) string {
	if pod.Name != "" {
		return pod.Name
	}
	if req.Name != "" {
		return req.Name
	}
	if pod.GenerateName != "" {
		return pod.GenerateName + "... (name not generated yet)"
	}
	return "" // should not happend
}

func getNamespace(pod *corev1.Pod, req *admissionv1beta1.AdmissionRequest) string {
	if pod.Namespace != "" {
		return pod.Namespace
	}
	if req.Namespace != "" {
		return req.Namespace
	}
	return "" // should not happend
}

func (h *Handler) mutatePodsFn(ctx context.Context, req atypes.Request, pod *corev1.Pod) error {
	saiNamesStr, ok := pod.Annotations[AnnotationKeyServiceAccountImportName]
	if !ok {
		return nil
	}

	ns := getNamespace(pod, req.AdmissionRequest)

	saiNames := strings.Split(saiNamesStr, ",")
	for _, saiName := range saiNames {
		sai := &v1alpha1.ServiceAccountImport{}
		if err := h.client.Get(ctx, types.NamespacedName{Namespace: ns, Name: saiName}, sai); err != nil {
			// throwing even when simply not found, to resolve race condition when pod and SAI are created concurrently
			// the controller trying to create the pod should retry later
			return fmt.Errorf("cannot find service account import %s in namespace %s", saiName, ns)
		}

		if len(sai.Status.Secrets) == 0 {
			// throwing to resolve race condition, idem above
			return fmt.Errorf(`service account import %s in namespace %s has no token, 
verify that the remote service account exists or retry when the secret has been created by the service account import controller`, saiName, ns)
		}

		secretName := sai.Status.Secrets[0].Name

		pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
			Name:         secretName,
			VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: secretName}},
		})

		for i := range pod.Spec.Containers {
			pod.Spec.Containers[i].VolumeMounts = append(pod.Spec.Containers[i].VolumeMounts,
				corev1.VolumeMount{
					Name:      secretName,
					ReadOnly:  true,
					MountPath: fmt.Sprintf("/var/run/secrets/admiralty.io/serviceaccountimports/%s", saiName)})
		}
	}

	return nil
}

// Handler implements inject.Client.
// A client will be automatically injected.
var _ inject.Client = &Handler{}

// InjectClient injects the client.
func (h *Handler) InjectClient(c client.Client) error {
	h.client = c
	return nil
}

// Handler implements inject.Decoder.
// A decoder will be automatically injected.
var _ inject.Decoder = &Handler{}

// InjectDecoder injects the decoder.
func (h *Handler) InjectDecoder(d atypes.Decoder) error {
	h.decoder = d
	return nil
}
