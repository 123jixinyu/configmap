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
	"configmap/domain/oss_client"
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	yamlV2 "gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"sync"
	"time"
)

var syncMap = &sync.Map{}

// ConfigMapWatcherReconciler reconciles a ConfigMapWatcher object
type ConfigMapWatcherReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	DynamicClient *dynamic.DynamicClient
}

//+kubebuilder:rbac:groups=configmap.xinyu.com,resources=configmapwatchers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=configmap.xinyu.com,resources=configmapwatchers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=configmap.xinyu.com,resources=configmapwatchers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=create;delete;deletecollection;get;list;patch;update;watch

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
	logger.Info("Own CRD Resource Reconcile", "name", configMapWatcher.Name, "namespace", configMapWatcher.Namespace)

	schedulerMapKey := getSchedulerKey(configMapWatcher.Name, configMapWatcher.Namespace)

	if storedScheduler, ok := syncMap.Load(schedulerMapKey); ok {
		storedScheduler.(*cron.Cron).Stop()
	}

	scheduler := cron.New()
	syncMap.Store(schedulerMapKey, scheduler)
	_, err := scheduler.AddFunc(configMapWatcher.Spec.Scheduler, func() {
		err := r.BackupToOss(ctx, configMapWatcher)
		if err != nil {
			logger.Info("backup to oss", "error", err.Error())
			return
		}
		logger.Info("backup to oss successful", "scheduler", configMapWatcher.Spec.Scheduler)
	})
	if err != nil {
		logger.Info("scheduler", "error", err.Error())
		syncMap.Delete(schedulerMapKey)
		return ctrl.Result{}, nil
	}
	scheduler.Start()

	return ctrl.Result{}, nil
}
func getSchedulerKey(name, namespace string) string {

	return fmt.Sprintf("%s.%s", namespace, name)
}
func (r *ConfigMapWatcherReconciler) BackupToOss(ctx context.Context, watcher configmapv1.ConfigMapWatcher) error {

	configMaps, err := r.GetConfigMapList(ctx)
	if err != nil {
		return err
	}

	directory := time.Now().Format("20060102150405")

	prefix := strings.TrimSpace(watcher.Spec.OssConfig.Directory)

	if prefix != "" {
		directory = prefix + "/" + directory
	}

	ossClient, err := oss_client.NewOssWithConfig(oss_client.OssConfig{
		EndPoint:        watcher.Spec.OssConfig.Endpoint,
		AccessKeyId:     watcher.Spec.OssConfig.AccessKey,
		AccessKeySecret: watcher.Spec.OssConfig.AccessSecret,
		Bucket:          watcher.Spec.OssConfig.Bucket,
	})
	for _, configMap := range configMaps {
		content, err := yamlV2.Marshal(configMap.Object)
		if err != nil {
			return err
		}
		if err := ossClient.PutObject(directory+"/"+configMap.GetNamespace()+"/"+configMap.GetName()+".yaml", content); err != nil {
			return err
		}
	}
	return nil
}
func (r *ConfigMapWatcherReconciler) GetConfigMapList(ctx context.Context) ([]unstructured.Unstructured, error) {

	list, err := r.DynamicClient.Resource(schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}).List(ctx, v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapWatcherReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&configmapv1.ConfigMapWatcher{}).Watches(&configmapv1.ConfigMapWatcher{}, &EventHandler{}).
		Complete(r)
}

type EventHandler struct {
	// Create is called in response to an create event - e.g. Pod Creation.
}

func (e EventHandler) Create(ctx context.Context, event event.CreateEvent, limitingInterface workqueue.RateLimitingInterface) {

}

func (e EventHandler) Update(ctx context.Context, event event.UpdateEvent, limitingInterface workqueue.RateLimitingInterface) {

}

func (e EventHandler) Delete(ctx context.Context, event event.DeleteEvent, limitingInterface workqueue.RateLimitingInterface) {

	logger := log.FromContext(ctx)
	watcher := event.Object.DeepCopyObject().(*configmapv1.ConfigMapWatcher)
	key := getSchedulerKey(watcher.Name, watcher.Namespace)

	scheduler, ok := syncMap.Load(key)
	if ok {
		scheduler.(*cron.Cron).Stop()
		syncMap.Delete(key)
		logger.Info("scheduler", "stop and delete", key)
	}
}

func (e EventHandler) Generic(ctx context.Context, event event.GenericEvent, limitingInterface workqueue.RateLimitingInterface) {

}
