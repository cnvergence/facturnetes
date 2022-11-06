package controllers

import (
	"context"

	facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"
	"github.com/cnvergence/facturnetes/pkg/resource"
	generator "github.com/cnvergence/invoice-generator/invoice"
	"gopkg.in/yaml.v2"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *InvoiceReconciler) ensureService(invoice *facturnetesv1.Invoice) error {
	svc := resource.Service(invoice)
	if err := ctrl.SetControllerReference(invoice, svc, r.Scheme); err != nil {
		return nil
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
	dep := resource.Deployment(invoice)
	if err := ctrl.SetControllerReference(invoice, dep, r.Scheme); err != nil {
		return nil
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
	sc := resource.Secret(invoice, pdf)
	if err := ctrl.SetControllerReference(invoice, sc, r.Scheme); err != nil {
		return nil
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
	ing := resource.Ingress(invoice)
	if err := ctrl.SetControllerReference(invoice, ing, r.Scheme); err != nil {
		return err
	}

	ingo := ing.DeepCopyObject().(*networkingv1.Ingress)
	op, err := ctrl.CreateOrUpdate(context.TODO(), r.client, ingo, func() error {
		ingo.Spec = ing.Spec
		return nil
	})
	if err != nil {
		r.log.Errorf("Could not create or patch the Ingress: %s", err)
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
