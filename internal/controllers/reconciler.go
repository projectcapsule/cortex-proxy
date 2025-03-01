package controllers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	capsulev1beta2 "github.com/projectcapsule/capsule/api/v1beta2"
	"github.com/projectcapsule/cortex-proxy/internal/metrics"
	"github.com/projectcapsule/cortex-proxy/internal/stores"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type TenantController struct {
	client.Client
	Metrics  *metrics.Recorder
	Scheme   *runtime.Scheme
	Store    *stores.TenantStore
	Log      logr.Logger
	Selector *metav1.LabelSelector
}

func (r *TenantController) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).For(&capsulev1beta2.Tenant{})

	// If a selector is provided, add an event filter so that only matching tenants trigger reconcile.
	if r.Selector != nil {
		selector, err := metav1.LabelSelectorAsSelector(r.Selector)
		if err != nil {
			return fmt.Errorf("invalid label selector: %w", err)
		}

		builder = builder.WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool {
				return selector.Matches(labels.Set(e.Object.GetLabels()))
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				return selector.Matches(labels.Set(e.ObjectNew.GetLabels()))
			},
			DeleteFunc: func(e event.DeleteEvent) bool {
				return selector.Matches(labels.Set(e.Object.GetLabels()))
			},
			GenericFunc: func(e event.GenericEvent) bool {
				return selector.Matches(labels.Set(e.Object.GetLabels()))
			},
		})
	}

	return builder.Complete(r)
}

func (r *TenantController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	origin := &capsulev1beta2.Tenant{}
	if err := r.Get(ctx, req.NamespacedName, origin); err != nil {
		r.lifecycle(&capsulev1beta2.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: req.Namespace,
			},
		})

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	r.Store.Update(origin)

	return ctrl.Result{}, nil
}

// First execttion of the controller to load the settings (without manager cache).
func (r *TenantController) Init(ctx context.Context, c client.Client) (err error) {
	tnts := &capsulev1beta2.TenantList{}

	var opts []client.ListOption

	// If a selector is provided, add it as a list option.
	if r.Selector != nil {
		selector, err := metav1.LabelSelectorAsSelector(r.Selector)
		if err != nil {
			return fmt.Errorf("invalid label selector: %w", err)
		}

		opts = append(opts, client.MatchingLabelsSelector{Selector: selector})
	}

	if err := c.List(ctx, tnts, opts...); err != nil {
		return fmt.Errorf("could not load tenants: %w", err)
	}

	for _, tnt := range tnts.Items {
		r.Store.Update(&tnt)
	}

	return
}

func (r *TenantController) lifecycle(tenant *capsulev1beta2.Tenant) {
	r.Store.Delete(&capsulev1beta2.Tenant{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tenant.Name,
			Namespace: tenant.Namespace,
		},
	})

	r.Metrics.DeleteMetricsForTenant(tenant)
}
