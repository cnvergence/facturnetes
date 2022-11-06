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
	"os"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	facturnetesv1 "github.com/cnvergence/facturnetes/api/v1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func init() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		os.Exit(1)
	}
	zap.ReplaceGlobals(logger)
}

//+kubebuilder:rbac:groups=facturnetes.cnvergence.io,resources=invoices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=facturnetes.cnvergence.io,resources=invoices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=facturnetes.cnvergence.io,resources=invoices/finalizers,verbs=update
//+kubebuilder:rbac:groups=*,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=secrets,verbs=get;list;watch;create;update;patch;delete

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
	invoice := facturnetesv1.Invoice{}
	r.log = zap.S().With("Invoice", req.NamespacedName)
	r.log.Info("Reconciling Invoice")

	if err := r.client.Get(ctx, req.NamespacedName, &invoice); err != nil {
		r.log.Error(err, "unable to fetch Invoice")
		return r.SetFailureStatus(ctx, &invoice, err)
	}

	pdf, err := r.generateInvoice(invoice)
	if err != nil {
		r.log.Error(err, "unable to generate PDF invoice")
		return r.SetFailureStatus(ctx, &invoice, err)
	}

	r.log.Debug("Ensuring that Secret exists")
	if err := r.ensureSecret(&invoice, pdf); err != nil {
		return r.SetFailureStatus(ctx, &invoice, err)
	}

	r.log.Debug("Ensuring that Deployment exists")
	if err := r.ensureDeployment(&invoice); err != nil {
		return r.SetFailureStatus(ctx, &invoice, err)
	}

	r.log.Debug("Ensuring that Service exists")
	if err := r.ensureService(&invoice); err != nil {
		return r.SetFailureStatus(ctx, &invoice, err)
	}

	if invoice.Spec.Ingress.Enabled {
		r.log.Debug("Ensuring that Ingress exists")
		if err := r.ensureIngress(&invoice); err != nil {
			return r.SetFailureStatus(ctx, &invoice, err)
		}
	}

	return r.SetSuccessStatus(ctx, &invoice)
}

// SetupWithManager sets up the controller with the Manager.
func (r *InvoiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		For(&facturnetesv1.Invoice{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Complete(r)
}

func (r *InvoiceReconciler) SetSuccessStatus(ctx context.Context, invoice *facturnetesv1.Invoice) (ctrl.Result, error) {
	invoice.Status.ObservedGeneration = invoice.Generation
	invoice.Status.LastProcessedTime = &metav1.Time{Time: time.Now()}
	invoice.Status.Phase = facturnetesv1.Success

	if err := r.client.Status().Update(ctx, invoice); err != nil {
		r.log.Error(err, "Unable to update the status")
		return ctrl.Result{
			RequeueAfter: 15 * time.Second,
		}, err
	}

	return ctrl.Result{}, nil
}

func (r *InvoiceReconciler) SetFailureStatus(ctx context.Context, invoice *facturnetesv1.Invoice, msg error) (ctrl.Result, error) {
	invoice.Status.Message = msg.Error()
	invoice.Status.ObservedGeneration = invoice.Generation
	invoice.Status.LastProcessedTime = &metav1.Time{Time: time.Now()}
	invoice.Status.Phase = facturnetesv1.Failure

	return ctrl.Result{
		RequeueAfter: 15 * time.Second,
	}, r.client.Status().Update(ctx, invoice)
}
