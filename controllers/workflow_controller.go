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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	wp5v1alpha1 "github.com/luis5tb/physics-workflow-operator/api/v1alpha1"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const workflowManifestFinalizer = "workflowmanifest/finalizer"
const KNATIVE_PLATFORM = "knative"
const OPENWHISK_PLATFORM = "openWhisk"

//+kubebuilder:rbac:groups=wp5.physics-faas.eu,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=wp5.physics-faas.eu,resources=workflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=wp5.physics-faas.eu,resources=workflows/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Workflow object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// 1. Get workflowManifest
	workflowManifest := &wp5v1alpha1.Workflow{}
	err := r.Get(ctx, req.NamespacedName, workflowManifest)
	if err != nil {
		if errors.IsNotFound(err) { // we deleted the CR object
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Check if the workflowManifest instance is marked to be deleted, which is indicated by the deletion timestamp being set.
	isWorkflowManifestMarkedToBeDeleted := workflowManifest.GetDeletionTimestamp() != nil
	if !isWorkflowManifestMarkedToBeDeleted && controllerutil.ContainsFinalizer(workflowManifest, workflowManifestFinalizer) {

		switch workflowManifest.Spec.Platform {
		case KNATIVE_PLATFORM:
			route, err := ReconcileKnativeResources(r, ctx, req.Namespace, workflowManifest)
			if err != nil {
				return ctrl.Result{}, err
			}
			err = UpdateKnativeWorkflowStatus(r, ctx, workflowManifest, route)
			if err != nil {
				return ctrl.Result{}, err
			}
		case OPENWHISK_PLATFORM:
			// TO DO
		default:
			// It must be stated, otherwise don't process
			return ctrl.Result{}, nil
		}
	}

	// Check if the workflowManifest instance is marked to be deleted, which is indicated by the deletion timestamp being set.
	if isWorkflowManifestMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(workflowManifest, workflowManifestFinalizer) {
			// Run finalization logic for workflowManifestFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeWorkflowManifest(ctx, req, workflowManifest); err != nil {
				return ctrl.Result{}, err
			}

			// Remove workflowManifest Finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(workflowManifest, workflowManifestFinalizer)
			err := r.Update(ctx, workflowManifest)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer for this CR
	if !controllerutil.ContainsFinalizer(workflowManifest, workflowManifestFinalizer) {
		controllerutil.AddFinalizer(workflowManifest, workflowManifestFinalizer)
		err := r.Update(ctx, workflowManifest)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *WorkflowReconciler) finalizeWorkflowManifest(ctx context.Context, req ctrl.Request, workflowManifest *wp5v1alpha1.Workflow) error {
	var logger = log.FromContext(ctx)

	switch workflowManifest.Spec.Platform {
	case KNATIVE_PLATFORM:
		err := CleanUpKnativeResources(r, ctx, workflowManifest)
		if err != nil {
			//panic(err)
			return err
		}
	case OPENWHISK_PLATFORM:
		// TO DO
	default:
		// No need to process
		return nil
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wp5v1alpha1.Workflow{}).
		Complete(r)
}
