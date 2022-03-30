package controllers

import (
	"github.com/oceanweave/operator-sdk-demo/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func NewDeploy(app *v1beta1.MyApp) *appsv1.Deployment {
	labels := map[string]string{"myapp": app.Name}
	seletor := &metav1.LabelSelector{
		MatchLabels: labels,
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deplyment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			// OwnerReferences 属于谁管理，宿主删除，此资源也删除
			// 为了 删除 myapp 时 将此 deployment 也删除
			// 此处逻辑与下面 service 共用，因此提取出来，形成一个公共函数
			OwnerReferences: makeOwnerRefer(app),
		},
		Spec: appsv1.DeploymentSpec{
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
		},
	}
}

func newContainers(app *v1beta1.MyApp) []corev1.Container {
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range app.Spec.Ports {
		containerPorts = append(containerPorts, corev1.ContainerPort{
			ContainerPort: svcPort.TargetPort.IntVal,
		})
	}
	return []corev1.Container{
		{
			Name:      app.Name,
			Image:     app.Spec.Image,
			Resources: app.Spec.Resources,
			Env:       app.Spec.Envs,
			Ports:     containerPorts,
		},
	}
}

func NewService(app *v1beta1.MyApp) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			// 为了 删除 myapp 时 将此 service 也删除
			OwnerReferences: makeOwnerRefer(app),
		},
		Spec: corev1.ServiceSpec{
			Ports: app.Spec.Ports,
			Type:  corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"myapp": app.Name,
			},
		},
	}
}

// 公共函数  表示属主是谁
func makeOwnerRefer(app *v1beta1.MyApp) []metav1.OwnerReference {
	return []metav1.OwnerReference{
		*metav1.NewControllerRef(app, schema.GroupVersionKind{
			Kind:    v1beta1.Kind,
			Group:   v1beta1.GroupVersion.Group,
			Version: v1beta1.GroupVersion.Version,
		}),
	}
}
