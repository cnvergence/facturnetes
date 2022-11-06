package resource

import facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"

func Labels(invoice *facturnetesv1.Invoice) map[string]string {
	return map[string]string{
		"app": invoice.Name,
	}
}
