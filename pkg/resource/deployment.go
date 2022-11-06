package resource

import (
	"fmt"

	facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	defaultDeploymentName = "viewer"
	defaultImageName      = "viewer:latest"
)

func Deployment(invoice *facturnetesv1.Invoice) *appsv1.Deployment {
	deploymentName := invoice.Spec.Deployment.Name
	if invoice.Spec.Deployment.Name == "" {
		deploymentName = defaultDeploymentName
	}
	imageName := invoice.Spec.Deployment.Image
	if invoice.Spec.Deployment.Image == "" {
		imageName = defaultImageName
	}

	labels := Labels(invoice)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    labels,
			Name:      invoice.Name,
			Namespace: invoice.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: fmt.Sprintf("%s-volume", deploymentName),
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: invoice.Name,
								Items: []corev1.KeyToPath{{
									Key:  "pdf",
									Path: "test.pdf",
								},
								},
							},
						}}},
					Containers: []corev1.Container{{
						Image: imageName,
						Name:  deploymentName,
						Env: []corev1.EnvVar{{
							Name:  "REQUEST_HASH",
							Value: "test.pdf",
						}},
						Ports: []corev1.ContainerPort{
							{
								ContainerPort: 3030,
								Name:          "http",
							},
						},
						ImagePullPolicy: invoice.Spec.Deployment.ImagePullPolicy,
						VolumeMounts: []corev1.VolumeMount{{
							Name:      fmt.Sprintf("%s-volume", deploymentName),
							MountPath: "/etc/config",
						},
						},
					}},
				},
			},
		},
	}
}

func Service(invoice *facturnetesv1.Invoice) *corev1.Service {
	labels := Labels(invoice)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      invoice.Name,
			Namespace: invoice.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       3030,
				TargetPort: intstr.FromString("http"),
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}
