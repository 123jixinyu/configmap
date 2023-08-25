/*
Copyright 2023.

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

package controller

import (
	configmapv1 "configmap/api/v1"
	"context"
	"fmt"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ConfigMapWatcherReconciler reconciles a ConfigMapWatcher object
type ConfigMapWatcherReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	DynamicClient *dynamic.DynamicClient
}

//+kubebuilder:rbac:groups=configmap.xinyu.com,resources=configmapwatchers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=configmap.xinyu.com,resources=configmapwatchers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=configmap.xinyu.com,resources=configmapwatchers/finalizers,verbs=update
//+kubebuilder:rbac:groups=*,resources=configmaps,verbs=create;delete;deletecollection;get;list;patch;update;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigMapWatcher object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.0/pkg/reconcile
func (r *ConfigMapWatcherReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	configMapWatcher := configmapv1.ConfigMapWatcher{}

	if err := r.Get(ctx, req.NamespacedName, &configMapWatcher); err != nil {
		return ctrl.Result{}, nil
	}

	logger.Info("Reconcile: " + configMapWatcher.Name)

	return ctrl.Result{}, nil
}

type MyEventHandler struct{}

func (*MyEventHandler) Create(ctx context.Context, createEvent event.CreateEvent, limitingInterface workqueue.RateLimitingInterface) {

	configMap := createEvent.Object.DeepCopyObject().(*coreV1.ConfigMap)

	fmt.Println(configMap.Name)
}

func (*MyEventHandler) Update(ctx context.Context, updateEvent event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {
}

func (*MyEventHandler) Delete(ctx context.Context, deleteEvent event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {

}

func (*MyEventHandler) Generic(ctx context.Context, genericEvent event.GenericEvent, limitingInterface workqueue.RateLimitingInterface) {

}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configmapv1.ConfigMapWatcher{}).Watches(&coreV1.ConfigMap{}, &MyEventHandler{}).
		Complete(r)
}
