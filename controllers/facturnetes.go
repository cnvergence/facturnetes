package controllers

import (
	"context"
	"net/url"
	"strings"

	facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"
	generator "github.com/cnvergence/invoice-generator/invoice"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *InvoiceReconciler) constructSecret(invoice *facturnetesv1.Invoice, data []byte) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      invoice.Name,
			Namespace: invoice.Namespace,
		},
		Data: map[string][]byte{"pdf": data},
	}

	if err := ctrl.SetControllerReference(invoice, secret, r.Scheme); err != nil {
		return nil, err
	}

	return secret, nil
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

func (r *InvoiceReconciler) constructIngress(invoice *facturnetesv1.Invoice) (*networkingv1.Ingress, error) {
	u, err := url.Parse(invoice.Spec.PublicURL)
	if err != nil {
		return &networkingv1.Ingress{}, err
	}

	tls := make([]networkingv1.IngressTLS, 0)

	if invoice.Spec.Ingress.TLSEnabled {
		if invoice.Spec.Ingress.TLSSecretName == "" {
			invoice.Spec.Ingress.TLSSecretName = u.Host + "-tls"
		}

		tls = []networkingv1.IngressTLS{
			{
				Hosts:      []string{u.Host},
				SecretName: invoice.Spec.Ingress.TLSSecretName,
			},
		}
	}
	pathType := networkingv1.PathTypePrefix

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      invoice.Name,
			Namespace: invoice.Namespace,
		},
		Spec: networkingv1.IngressSpec{
			TLS: tls,
			Rules: []networkingv1.IngressRule{
				{
					Host: u.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/" + strings.TrimPrefix(u.Path, "/"),
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: invoice.Name,
											Port: networkingv1.ServiceBackendPort{
												Name: "https",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if err := ctrl.SetControllerReference(invoice, ing, r.Scheme); err != nil {
		return nil, err
	}

	return ing, nil
}

func (r *InvoiceReconciler) constructDeployment(invoice *facturnetesv1.Invoice) (*appsv1.Deployment, error) {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"app": invoice.Name,
			},
			Name:      invoice.Name,
			Namespace: invoice.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": invoice.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": invoice.Name,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{{
						Name: "viewer-volume",
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
			},
		},
	}

	if err := ctrl.SetControllerReference(invoice, dep, r.Scheme); err != nil {
		return nil, err
	}

	return dep, nil
}

func (r *InvoiceReconciler) ensureService(invoice *facturnetesv1.Invoice) error {
	svc, err := r.constructService(invoice)
	if err != nil {
		r.log.Error(err, "unable to construct Service from template")
		return err
	}
	svco := svc.DeepCopyObject().(*corev1.Service)
	op, err := ctrl.CreateOrUpdate(context.TODO(), r.client, svc, func() error {
		svco.Spec = svc.Spec
		return nil
	})
	if err != nil {
		r.log.Errorf("Could not create or patch the service: %s", err)
		return err
	}

	r.log.Infow("Create/Update operation succeeded", "operation", op)

	return nil
}

func (r *InvoiceReconciler) ensureDeployment(invoice *facturnetesv1.Invoice) error {
	dep, err := r.constructDeployment(invoice)
	if err != nil {
		r.log.Error(err, "unable to construct Deployment from template")
		return err
	}
	depo := dep.DeepCopyObject().(*appsv1.Deployment)
	op, err := ctrl.CreateOrUpdate(context.TODO(), r.client, depo, func() error {
		depo.Spec = dep.Spec
		return nil
	})
	if err != nil {
		r.log.Errorf("Could not create or patch the Deployment: %s", err)
		return err
	}
	r.log.Infow("Create/Update operation succeeded", "operation", op)

	return nil
}

func (r *InvoiceReconciler) ensureSecret(invoice *facturnetesv1.Invoice, pdf []byte) error {
	sc, err := r.constructSecret(invoice, pdf)
	if err != nil {
		r.log.Error(err, "unable to construct Secret from template")
		return err
	}
	sco := sc.DeepCopyObject().(*corev1.Secret)
	r.log.Infow("Reconciling Secret", "name", sco.Name)
	op, err := ctrl.CreateOrUpdate(context.TODO(), r.client, sco, func() error {
		sco.Data = sc.Data
		return nil
	})
	if err != nil {
		r.log.Errorf("Could not create or patch the Secret: %s", err)
		return err
	}

	r.log.Infow("Create/Update operation succeeded", "operation", op)

	return nil
}

func (r *InvoiceReconciler) ensureIngress(invoice *facturnetesv1.Invoice) error {
	ing, err := r.constructIngress(invoice)
	if err != nil {
		r.log.Error(err, "unable to construct Deployment from template")
		return err
	}
	ingo := ing.DeepCopyObject().(*networkingv1.Ingress)
	op, err := ctrl.CreateOrUpdate(context.TODO(), r.client, ingo, func() error {
		ingo.Spec = ing.Spec
		return nil
	})
	if err != nil {
		r.log.Errorf("Could not create or patch the Deployment: %s", err)
		return err
	}
	r.log.Infow("Create/Update operation succeeded", "operation", op)

	return nil
}

func (r *InvoiceReconciler) generateInvoice(invoice facturnetesv1.Invoice) ([]byte, error) {
	data, err := yaml.Marshal(invoice.Spec.InvoiceData)
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
