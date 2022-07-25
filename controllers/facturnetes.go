package controllers

import (
	"context"

	facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"
	generator "github.com/cnvergence/invoice-generator/invoice"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *InvoiceReconciler) constructConfigMap(invoice *facturnetesv1.Invoice, data []byte) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        invoice.Name,
			Namespace:   invoice.Namespace,
		},
		BinaryData: map[string][]byte{"pdf": data},
	}

	if err := ctrl.SetControllerReference(invoice, configMap, r.Scheme); err != nil {
		return nil, err
	}

	return configMap, nil
}

func (r *InvoiceReconciler) constructService(invoice *facturnetesv1.Invoice) (*corev1.Service, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      invoice.Name,
			Namespace: invoice.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": invoice.Name,
			},
			Ports: []corev1.ServicePort{{
				Protocol:   corev1.ProtocolTCP,
				Port:       3030,
				TargetPort: intstr.FromInt(3030),
			}},
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	if err := ctrl.SetControllerReference(invoice, svc, r.Scheme); err != nil {
		return nil, err
	}

	return svc, nil
}

func (r *InvoiceReconciler) constructPod(invoice *facturnetesv1.Invoice) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": invoice.Name,
			},
			Annotations: make(map[string]string),
			Name:        invoice.Name,
			Namespace:   invoice.Namespace,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name: "viewer-volume",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: invoice.Name,
						},
						Items: []corev1.KeyToPath{{
							Key:  "pdf",
							Path: "test.pdf",
						},
						},
					},
				}}},
			Containers: []corev1.Container{{
				Image: "viewer:latest",
				Name:  "viewer",
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
				ImagePullPolicy: "Never",
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "viewer-volume",
					MountPath: "/etc/config",
				},
				},
			}},
		},
	}

	if err := ctrl.SetControllerReference(invoice, pod, r.Scheme); err != nil {
		return nil, err
	}

	return pod, nil
}

func (r *InvoiceReconciler) ensureService(invoice *facturnetesv1.Invoice) error {
	svc, err := r.constructService(invoice)
	if err != nil {
		r.log.Error(err, "unable to construct Pod from template")
		return err
	}

	r.log.Infof("creating a new Pod %s/%s", svc.Namespace, svc.Name)
	if err := r.client.Create(context.TODO(), svc); err != nil {
		r.log.Errorf("unable to create Pod: %v", err)
		return err
	}

	return nil
}

func (r *InvoiceReconciler) ensurePod(invoice *facturnetesv1.Invoice) error {
	pod, err := r.constructPod(invoice)
	if err != nil {
		r.log.Error(err, "unable to construct Pod from template")
		return err
	}

	r.log.Infof("creating a new Pod %s/%s", pod.Namespace, pod.Name)
	if err := r.client.Create(context.TODO(), pod); err != nil {
		r.log.Errorf("unable to create Pod: %v", err)
		return err
	}

	return nil
}

func (r *InvoiceReconciler) ensureConfigMap(invoice *facturnetesv1.Invoice, pdf []byte) error {
	configMap, err := r.constructConfigMap(invoice, pdf)
	if err != nil {
		r.log.Error(err, "unable to construct ConfigMap from template")
		return err
	}

	r.log.Infof("creating a new ConfigMap %s/%s", configMap.Namespace, configMap.Name)
	if err := r.client.Create(context.TODO(), configMap); err != nil {
		r.log.Errorf("unable to create ConfigMap: %v", err)
		return err
	}

	return nil
}

func (r *InvoiceReconciler) generateInvoice(invoice facturnetesv1.Invoice) ([]byte, error) {
	data, err := yaml.Marshal(invoice.Spec)
	if err != nil {
		r.log.Error(err, "unable to marshal yaml")
		return nil, err
	}
	inv, err := generator.New(data)
	if err != nil {
		r.log.Error(err, "unable to create invoice")
		return nil, err
	}
	bytes, err := inv.SaveAsBytes()
	if err != nil {
		r.log.Error(err, "unable to create invoice")
		return nil, err
	}

	return bytes, nil
}
