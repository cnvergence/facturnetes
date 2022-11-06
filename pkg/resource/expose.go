package resource

import (
	"net/url"
	"strings"

	facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Ingress(invoice *facturnetesv1.Invoice) *networkingv1.Ingress {
	u, err := url.Parse(invoice.Spec.Exposure.PublicURL)
	if err != nil {
		return &networkingv1.Ingress{}
	}

	tls := make([]networkingv1.IngressTLS, 0)

	if invoice.Spec.Exposure.Ingress.TLSEnabled {
		if invoice.Spec.Exposure.Ingress.TLSSecretName == "" {
			invoice.Spec.Exposure.Ingress.TLSSecretName = u.Host + "-tls"
		}

		tls = []networkingv1.IngressTLS{
			{
				Hosts:      []string{u.Host},
				SecretName: invoice.Spec.Exposure.Ingress.TLSSecretName,
			},
		}
	}
	pathType := networkingv1.PathTypeImplementationSpecific

	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      invoice.Name,
			Namespace: invoice.Namespace,
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: &invoice.Spec.Exposure.Ingress.IngressClassName,
			TLS:              tls,
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
												Number: 3030,
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
}
