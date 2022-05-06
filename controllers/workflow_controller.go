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

	wp5v1alpha1 "gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/api/v1alpha1"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const workflowManifestFinalizer = "workflowmanifest/finalizer"

//+kubebuilder:rbac:groups=wp5.physics-faas.eu,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=wp5.physics-faas.eu,resources=workflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=wp5.physics-faas.eu,resources=workflows/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

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
	var logger = log.FromContext(ctx)
	//var PHYSICS_OW_PROXY_NAME string = lookupEnv("PHYSICS_OW_PROXY_NAME", "physics-ow-proxy")
	// TODO(user): your logic here
	logger.Info("Reconcile() => Getting Workflow Manifest...")
	workflowManifest := &wp5v1alpha1.Workflow{}
	err := r.Get(ctx, req.NamespacedName, workflowManifest)
	//fmt.Println(workflowManifest)
	//logger.Info(fmt.Sprint(workflowManifest))
	if err != nil {
		if errors.IsNotFound(err) { // we deleted the CR object
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Something was wrong here!")
		return ctrl.Result{}, err
	}

	/*
		// Check if the deployment already exists, if not create a new one
		foundDep := &appsv1.Deployment{}
		err = r.Get(ctx, types.NamespacedName{Name: PHYSICS_OW_PROXY_NAME, Namespace: workflowManifest.Namespace}, foundDep)
		if err != nil && errors.IsNotFound(err) {
			// Define a new deployment
			dep := CreateDeploymentForOpenWhiskProxy(workflowManifest)
			logger.Info("Creating a new OpenWhisk Proxy deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			err = r.Create(ctx, dep)
			if err != nil {
				logger.Error(err, "Failed to create OpenWhisk Proxy deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
				return ctrl.Result{}, err
			}
			// Deployment created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			logger.Error(err, "Failed to get OpenWhisk Proxy deployment")
			return ctrl.Result{}, err
		}
		// Check if the service already exists, if not create a new one
		foundSvc := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{Name: PHYSICS_OW_PROXY_NAME, Namespace: workflowManifest.Namespace}, foundSvc)
		if err != nil && errors.IsNotFound(err) {
			// Define a new service
			svc := CreateServiceForOpenWhiskProxy(workflowManifest)
			logger.Info("Creating a new OpenWhisk Proxy service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			err = r.Create(ctx, svc)
			if err != nil {
				logger.Error(err, "Failed to create OpenWhisk Proxy service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
				return ctrl.Result{}, err
			}
			// Service created successfully - return and requeue
			return ctrl.Result{Requeue: true}, nil
		} else if err != nil {
			logger.Error(err, "Failed to get OpenWhisk Proxy service")
			return ctrl.Result{}, err
		}
	*/

	// Check if the workflowManifest instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isWorkflowManifestMarkedToBeDeleted := workflowManifest.GetDeletionTimestamp() != nil

	if !isWorkflowManifestMarkedToBeDeleted &&
		controllerutil.ContainsFinalizer(workflowManifest, workflowManifestFinalizer) {
		err := UpdateExternalResources(logger, req.Namespace, workflowManifest)
		if err != nil {
			//panic(err)
			return ctrl.Result{}, err
		}
		//return ctrl.Result{}, nil
	}

	// Check if the workflowManifest instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
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
	logger.Info("Reconcile() end.")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wp5v1alpha1.Workflow{}).
		//Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *WorkflowReconciler) finalizeWorkflowManifest(ctx context.Context, req ctrl.Request, workflowManifest *wp5v1alpha1.Workflow) error {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	var logger = log.FromContext(ctx)
	logger.Info("finalizeWorkflowManifest()...")
	err := CleanUpExternalResources(logger, req.Namespace, workflowManifest)
	if err != nil {
		//panic(err)
		return err
	}
	logger.Info("Successfully finalized workflowManifest")
	return nil
}
