/*
Copyright 2022 dfy.

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

package controllers

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	appv1beta1 "github.com/oceanweave/operator-sdk-demo/api/v1beta1"
)

var (
	oldSpecAnnotation = "old/spec"
)

// MyAppReconciler reconciles a MyApp object
type MyAppReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// 赋予该资源及资源的 rbac 权限  看 增加了 deployment 和 service  两行
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.oceanweave.io,resources=myapps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=app.oceanweave.io,resources=myapps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=app.oceanweave.io,resources=myapps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyApp object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *MyAppReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log2 := log.FromContext(ctx, "myapp", req.NamespacedName)
	//log2 := r.Log.WithValues("myapp", req.NamespacedName)
	// TODO(user): your logic here
	// 首先获取 MyApp 实例
	var myapp appv1beta1.MyApp
	err := r.Client.Get(ctx, req.NamespacedName, &myapp)
	if err != nil {
		//// 在删除一个不存在的对象时，可能会包 not found 错误
		//// 这种情况不需要重新入队列排队修复
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 上面的劣势：当 deployment 或 service 被误删时，MyApp 并不会重新创建
	// 调谐，获取当前的状态，与期望状态进行比较
	var deploy appsv1.Deployment
	deploy.Name = myapp.Name
	deploy.Namespace = myapp.Namespace
	log2.Info("开始调谐。。。。。创建资源")
	or, err := ctrl.CreateOrUpdate(ctx, r.Client, &deploy, func() error {
		// 调谐
		log2.Info("创建deploy资源")
		MutateDeployment(&myapp, &deploy)
		return controllerutil.SetControllerReference(&myapp, &deploy, r.Scheme)
	})
	if err != nil {
		log2.Info("创建deploy失败")
		return ctrl.Result{}, err
	}
	log2.Info("CreateOrUpdate", "Deployment", or)
	var svc corev1.Service
	svc.Namespace = myapp.Namespace
	svc.Name = myapp.Name
	svcor, err := ctrl.CreateOrUpdate(ctx, r.Client, &svc, func() error {
		// 调谐
		MutateService(&myapp, &svc)
		return controllerutil.SetControllerReference(&myapp, &svc, r.Scheme)
		// 注意下面的错误， SetOwnerReference 不具备 watch 机制，因此不会进行调谐
		//return controllerutil.SetOwnerReference(&myapp, &svc, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	log2.Info("CreateOrUpdate", "Service", svcor)
	return ctrl.Result{}, nil
}
func MutateService(app *appv1beta1.MyApp, svc *corev1.Service) {
	svc.Spec = corev1.ServiceSpec{
		ClusterIP: svc.Spec.ClusterIP,
		Ports:     app.Spec.Ports,
		Type:      corev1.ServiceTypeNodePort,
		Selector: map[string]string{
			"myapp": app.Name,
		},
	}
}
func MutateDeployment(app *appv1beta1.MyApp, deploy *appsv1.Deployment) {
	labels := map[string]string{"myapp": app.Name}
	seletor := &metav1.LabelSelector{
		MatchLabels: labels,
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: app.Spec.Size,
		Selector: seletor, // 匹配 pod label
		Template: corev1.PodTemplateSpec{ // pod 模板定义
			ObjectMeta: metav1.ObjectMeta{
				Labels: labels,
			},
			Spec: corev1.PodSpec{
				Containers: newContainers(app),
			},
		},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyAppReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appv1beta1.MyApp{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
