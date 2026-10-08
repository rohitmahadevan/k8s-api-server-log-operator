package controllers

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/rohitmahadevan/k8s-api-server-log-operator/api/v1alpha1"
)

type APIServerLogReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	KubeClient kubernetes.Interface
}

func (r *APIServerLogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var instance v1alpha1.APIServerLog
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	ns := instance.Spec.Namespace
	if ns == "" {
		ns = "kube-system"
	}

	container := instance.Spec.Container
	if container == "" {
		container = "kube-apiserver"
	}

	tailLines := int64(200)
	if instance.Spec.TailLines != nil {
		tailLines = *instance.Spec.TailLines
	}

	selector := labels.SelectorFromSet(labels.Set{
		"component": "kube-apiserver",
		"tier":      "control-plane",
	})

	podList := &corev1.PodList{}
	if err := r.List(ctx, podList, client.InNamespace(ns), &client.ListOptions{LabelSelector: selector}); err != nil {
		logger.Error(err, "failed to list kube-apiserver pods")
		return ctrl.Result{}, err
	}

	if len(podList.Items) == 0 {
		err := fmt.Errorf("no kube-apiserver pod found in namespace %s", ns)
		logger.Error(err, "api-server pod not found")
		return ctrl.Result{}, err
	}

	pod := podList.Items[0]

	logsReq := r.KubeClient.CoreV1().Pods(ns).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: container,
		TailLines: &tailLines,
	})

	stream, err := logsReq.Stream(ctx)
	if err != nil {
		instance.Status.LastError = err.Error()
		instance.Status.LastFetched = metav1.NewTime(time.Now())
		if updateErr := r.Status().Update(ctx, &instance); updateErr != nil {
			logger.Error(updateErr, "failed to update status after log stream error")
		}
		return ctrl.Result{}, err
	}
	defer stream.Close()

	if _, err := io.Copy(os.Stdout, stream); err != nil {
		instance.Status.LastError = err.Error()
		instance.Status.LastFetched = metav1.NewTime(time.Now())
		if updateErr := r.Status().Update(ctx, &instance); updateErr != nil {
			logger.Error(updateErr, "failed to update status after copy error")
		}
		return ctrl.Result{}, err
	}

	instance.Status.LastError = ""
	instance.Status.LastFetched = metav1.NewTime(time.Now())
	if err := r.Status().Update(ctx, &instance); err != nil {
		logger.Error(err, "failed to update APIServerLog status")
		return ctrl.Result{}, err
	}

	logger.Info("fetched kube-apiserver logs", "pod", pod.Name, "namespace", ns, "container", container)
	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *APIServerLogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.APIServerLog{}).
		Complete(r)
}
