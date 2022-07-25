/*
Copyright 2022.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// InvoiceReconciler reconciles a Invoice object
type InvoiceReconciler struct {
	client client.Client
	Scheme *runtime.Scheme
	log    *zap.SugaredLogger
}

func NewReconciler(mgr manager.Manager) *InvoiceReconciler {
	return &InvoiceReconciler{
		client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
		log:    zap.S(),
	}
}

//+kubebuilder:rbac:groups=facturnetes.cnvergence.io,resources=invoices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=facturnetes.cnvergence.io,resources=invoices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=facturnetes.cnvergence.io,resources=invoices/finalizers,verbs=update
//+kubebuilder:rbac:groups=*,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Invoice object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *InvoiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	invoice := &facturnetesv1.Invoice{}

	if err := r.client.Get(ctx, req.NamespacedName, invoice); err != nil {
		r.log.Error(err, "unable to fetch Invoice")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	pdf, err := r.generateInvoice(*invoice)
	if err != nil {
		r.log.Error(err, "unable to generate PDF invoice")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}

	configMap, err := r.constructConfigMap(invoice, pdf)
	if err != nil {
		r.log.Error(err, "unable to construct config map from template")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}

	pod, err := r.constructPod(invoice)
	if err != nil {
		r.log.Error(err, "unable to construct pod from template")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}
	svc, err := r.constuctService(invoice)
	if err != nil {
		r.log.Error(err, "unable to construct service from template")
		// don't bother requeuing until we get a change to the spec
		return ctrl.Result{}, nil
	}
	// ...and create it on the cluster
	if err := r.client.Create(ctx, configMap); err != nil {
		r.log.Error(err, "unable to create ConfigMap", configMap)
		return ctrl.Result{}, err
	}

	// ...and create it on the cluster
	if err := r.client.Create(ctx, pod); err != nil {
		r.log.Error(err, "unable to create Pod", pod)
		return ctrl.Result{}, err
	}

	if err := r.client.Create(ctx, svc); err != nil {
		r.log.Error(err, "unable to create Service", svc)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InvoiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&facturnetesv1.Invoice{}).
		Complete(r)
}
