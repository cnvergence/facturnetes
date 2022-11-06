package resource

import (
	facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Secret(invoice *facturnetesv1.Invoice, data []byte) *corev1.Secret {
	labels := Labels(invoice)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      invoice.Name,
			Namespace: invoice.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{"pdf": data},
	}
}
