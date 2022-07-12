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
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		r.UpdateWorkflowStatus(ctx, workflowManifest)
		//err2 := r.UpdateWorkflowStatus(ctx, workflowManifest)
		//if err2 != nil {
		//	return ctrl.Result{}, err2
		//}
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
} // Reconcile()

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wp5v1alpha1.Workflow{}).
		//Owns(&appsv1.Deployment{}).
		Complete(r)
} // SetupWithManager()

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
} // finalizeWorkflowManifest()

func (r *WorkflowReconciler) UpdateWorkflowStatus(ctx context.Context, workflowManifest *wp5v1alpha1.Workflow) error {
	var logger = log.FromContext(ctx)
	logger.Info("UpdateWorkflowStatus()...")
	var condition metav1.Condition
	var conditions []metav1.Condition
	//Update workflow status
	conditions = workflowManifest.Status.Conditions
	if len(conditions) == 0 {
		condition = metav1.Condition{
			Type:               "Applied",
			Status:             "False",
			ObservedGeneration: workflowManifest.ObjectMeta.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Unknown",
			Message:            "Pending to apply",
		}
		conditions = append(conditions, condition)
		condition = metav1.Condition{
			Type:               "Available",
			Status:             "False",
			ObservedGeneration: workflowManifest.ObjectMeta.Generation,
			LastTransitionTime: metav1.Time{Time: time.Now()},
			Reason:             "Unknown",
			Message:            "Not ready",
		}
		conditions = append(conditions, condition)
	} // len(conditions) == 0
	var stop bool = false
	var appliedCount int = 0
	var availableCount int = 0
	for k := range conditions {
		conditions[k].Status = "False"
		conditions[k].Reason = "Unknown"
		conditions[k].Message = ""
	}
	//for idx, action := range workflowManifest.Spec.Actions {
	for idx := range workflowManifest.Spec.Actions {
		if stop {
			break
		}
		//UpdateActionStatus(workflowManifest, &action, "")
		switch workflowManifest.Status.ActionStatuses[idx].State {
		case "Error":
			stop = true
		case "Applied":
			appliedCount++
		case "Available":
			availableCount++
		default: // Unknown
		} // switch workflowManifest.Status.ActionStatuses[idx].State
	} // range workflowManifest.Spec.Actions
	actionCount := len(workflowManifest.Spec.Actions)
	if actionCount == appliedCount+availableCount {
		for k := range conditions {
			if conditions[k].Type == "Applied" {
				conditions[k].Status = "True"
				conditions[k].Reason = "Registered"
				conditions[k].Message = "Actions registered"
			} else {
				if conditions[k].Type == "Available" {
					conditions[k].Status = "False"
					conditions[k].Reason = "Unready"
					conditions[k].Message = "Waiting availability of remote actions"
					if actionCount == availableCount {
						conditions[k].Status = "True"
						conditions[k].Reason = "Ready"
						conditions[k].Message = "Ready to serving"
					}
				}
			}
			conditions[k].LastTransitionTime = metav1.Time{Time: time.Now()}
			conditions[k].ObservedGeneration = workflowManifest.ObjectMeta.Generation
		}
	}
	workflowManifest.Status.Conditions = conditions
	actionStatSer, err := json.Marshal(workflowManifest.Status.ActionStatuses)
	if err == nil {
		workflowManifest.Status.ActionStatSer = string(actionStatSer)
	}
	err = r.Status().Update(ctx, workflowManifest)
	if err != nil {
		logger.Error(err, "Failed to update Workflow status")
		//return ctrl.Result{}, err
		return err
	}
	logger.Info("UpdateWorkflowStatus() end.")
	return nil
} // UpdateWorkflowStatus()

func UpdateActionStatus(workflowManifest *wp5v1alpha1.Workflow, action *wp5v1alpha1.Action, message string) {
	//var PHYSICS_ACTION_PROXY_IMAGE string = lookupEnv("PHYSICS_ACTION_PROXY_IMAGE", "action-proxy")
	var PHYSICS_ACTION_PROXY_PARAM string = lookupEnv("PHYSICS_ACTION_PROXY_PARAM", "backendURL")
	var found bool = false
	var idx int = -1
	var actionStatus wp5v1alpha1.ActionStatus
	for k := range workflowManifest.Status.ActionStatuses {
		if workflowManifest.Status.ActionStatuses[k].Id == action.Id {
			found = true
			idx = k
		}
	}
	namespace := workflowManifest.Namespace
	if namespace == "default" {
		namespace = "guest"
	}
	if found { // check for updates like version
		actionStatus = workflowManifest.Status.ActionStatuses[idx]
		actionStatus.Name = action.Name
		actionStatus.Namespace = namespace + "/" + workflowManifest.Name
		actionStatus.Version = action.Version
		actionStatus.Runtime = action.Runtime
		actionStatus.State = "Unknown"
		actionStatus.Message = ""
		actionStatus.BackendURL = ""
		actionStatus.Remote = ""
	} else { // new or incremental update
		actionStatus = wp5v1alpha1.ActionStatus{
			Name:       action.Name,
			Namespace:  namespace + "/" + workflowManifest.Name,
			Id:         action.Id,
			Version:    action.Version,
			Runtime:    action.Runtime,
			State:      "Unknown",
			Message:    "",
			BackendURL: "",
			Remote:     "",
		}
		workflowManifest.Status.ActionStatuses = append(workflowManifest.Status.ActionStatuses, actionStatus)
		idx = len(workflowManifest.Status.ActionStatuses) - 1
	}
	var actionError bool = false
	if len(message) > 0 {
		actionError = true
	}
	if !actionError { // Error registering function
		actionStatus.State = "Applied" // Registered
		actionStatus.Message = ""
		if _, ok := action.Annotations["remote"]; ok { // Remote function
			actionStatus.Remote = "true"
			actionStatus.BackendURL = "" // For action updates to remote
			//if action.Runtime == "blackbox" && action.Image == PHYSICS_ACTION_PROXY_IMAGE { // Remote function
			if input, ok := action.FunctionInput[PHYSICS_ACTION_PROXY_PARAM]; ok {
				if len(input.Value) > 0 {
					actionStatus.State = "Available" // Ready to serving
					actionStatus.BackendURL = input.Value
				}
			}
		} else { // Local function
			actionStatus.State = "Available" // Ready to serving
			actionStatus.BackendURL = setBackendURL(&actionStatus, workflowManifest.Annotations["cluster"], action.Annotations["api-id"])
		}
	} else {
		actionStatus.State = "Error"
		actionStatus.Message = message
	}
	workflowManifest.Status.ActionStatuses[idx] = actionStatus
} // UpdateActionStatus()

func setBackendURL(actionStatus *wp5v1alpha1.ActionStatus, cluster string, apiID string) string {
	var result string
	var PHYSICS_BACKENDURL_PATTERN string = lookupEnv("PHYSICS_BACKENDURL_PATTERN",
		"http://@REMOTE-CLUSTER-sub-ow.openwhisk.svc.clusterset.local/api/v1/web/@NAMESPACE/@PACKAGE/@ACTION.json") // OW Web url through by submariner (.json)
	//	"http://@REMOTE-CLUSTER-sub-ow.openwhisk.svc.clusterset.local/api/@API-ID/@PACKAGE/@ACTION")		// OW API Gateway url through submariner
	//var PHYSICS_APIGATEWAY_BASEURL string = lookupEnv("PHYSICS_APIGATEWAY_BASEURL",
	//		"http://" + cluster + "-sub-ow.openwhisk.svc.clusterset.local:8080/api/")
	namespace := strings.Split(actionStatus.Namespace, "/")[0] // "namespace/package"
	namespace = strings.Replace(url.QueryEscape(namespace), "+", "%20", -1)
	pckg := strings.Split(actionStatus.Namespace, "/")[1] // "namespace/package"
	pckg = strings.Replace(url.QueryEscape(pckg), "+", "%20", -1)
	action := strings.Replace(url.QueryEscape(actionStatus.Name), "+", "%20", -1)
	//result = PHYSICS_APIGATEWAY_BASEURL + apiID + "/" + pckg + "/" + name
	result = strings.ReplaceAll(PHYSICS_BACKENDURL_PATTERN, "@REMOTE-CLUSTER", cluster)
	result = strings.ReplaceAll(result, "@API-ID", apiID)
	result = strings.ReplaceAll(result, "@NAMESPACE", namespace) // Same as apiID?
	result = strings.ReplaceAll(result, "@PACKAGE", pckg)
	result = strings.ReplaceAll(result, "@ACTION", action)
	return result
}
