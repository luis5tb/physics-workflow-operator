// Copyright 2021 Atos
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Created on 23 Mar 2022
// Updated on 23 Mar 2022
//
// @author: ATOS
package controllers

import (
	"context"
	"encoding/json"
	"fmt"
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
	logow "gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/common/logs"
)

// path used in logs
const pathLOG string = "## [PHYSICS-WORKFLOW-OPERATOR] [controllers] "

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

/**
 * Reconcile is part of the main kubernetes reconciliation loop which aims to
 * move the current state of the cluster closer to the desired state.
 * TODO(user): Modify the Reconcile function to compare the state specified by
 * the Workflow object against the actual cluster state, and then
 * perform operations to make the cluster state reflect the state specified by
 * the user.
 *
 * For more details, check Reconcile and its Result here:
 * - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.10.0/pkg/reconcile
 */
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logow.Info(pathLOG + "[Reconcile] #################### Reconciliation loop ####################")
	logow.Info(pathLOG + "[Reconcile] 1. Getting Workflow Manifest...")

	var logger = log.FromContext(ctx)

	// 1. Get workflowManifest
	workflowManifest := &wp5v1alpha1.Workflow{}
	err := r.Get(ctx, req.NamespacedName, workflowManifest)
	if err != nil {
		if errors.IsNotFound(err) { // we deleted the CR object
			return ctrl.Result{}, nil
		}
		logow.Error(pathLOG + "[Reconcile] Error getting Workflow manifest: " + err.Error())
		return ctrl.Result{}, err
	}

	// 2. Check if the workflowManifest instance is marked to be deleted, which is indicated by the deletion timestamp being set.
	logow.Info(pathLOG + "[Reconcile] 2. Checking if Workflow Manifest is marked to be deleted ...")
	isWorkflowManifestMarkedToBeDeleted := workflowManifest.GetDeletionTimestamp() != nil

	if !isWorkflowManifestMarkedToBeDeleted && controllerutil.ContainsFinalizer(workflowManifest, workflowManifestFinalizer) {
		logow.Debug(pathLOG + "[Reconcile] Workflow Manifest is NOT marked to be deleted. Updating External resources and Workflow status ...")

		err := UpdateExternalResources(logger, req.Namespace, workflowManifest)
		if err != nil {
			logow.Error(pathLOG+"[Reconcile] Failed to update external resources: ", err)
		}

		err = r.UpdateWorkflowStatus(ctx, workflowManifest)
		if err != nil {
			logow.Error(pathLOG+"[Reconcile] Failed to update workflow status: ", err)
			return ctrl.Result{}, err
		}
	}

	// Check if the workflowManifest instance is marked to be deleted, which is indicated by the deletion timestamp being set.
	if isWorkflowManifestMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(workflowManifest, workflowManifestFinalizer) {
			// Run finalization logic for workflowManifestFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeWorkflowManifest(ctx, req, workflowManifest); err != nil {
				logow.Error(pathLOG+"[Reconcile] Failed to finalize workflow manifest: ", err)
				return ctrl.Result{}, err
			}

			// Remove workflowManifest Finalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(workflowManifest, workflowManifestFinalizer)
			err := r.Update(ctx, workflowManifest)
			if err != nil {
				logow.Error(pathLOG+"[Reconcile] Failed to remove workflow manifest: ", err)
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
			logow.Error(pathLOG+"[Reconcile] Failed to add finalizer: ", err)
			return ctrl.Result{}, err
		}
	}
	logow.Info(pathLOG + "[Reconcile] Reconcile() end.")

	//time.Sleep(5 * time.Second)

	return ctrl.Result{}, nil
}

/**
 * SetupWithManager sets up the controller with the Manager.
 */
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	logow.Debug(pathLOG + "[SetupWithManager] 'sigs.k8s.io/controller-runtime' Manager values: ")
	logow.Debug(pathLOG + "[SetupWithManager] Host:       " + mgr.GetConfig().Host)
	logow.Debug(pathLOG + "[SetupWithManager] APIPath:    " + mgr.GetConfig().APIPath)
	logow.Debug(pathLOG + "[SetupWithManager] ServerName: " + mgr.GetConfig().ServerName)
	logow.Debug(pathLOG + "[SetupWithManager] Username:   " + mgr.GetConfig().Username)
	logow.Debug(pathLOG + "[SetupWithManager] UserAgent:  " + mgr.GetConfig().UserAgent)

	fmt.Printf("'sigs.k8s.io/controller-runtime' Manager: %v\n-------------------\n", mgr)

	return ctrl.NewControllerManagedBy(mgr).
		For(&wp5v1alpha1.Workflow{}).
		Complete(r)
}

/**
 * finalizeWorkflowManifest
 * TODO(user): Add the cleanup steps that the operator needs to do before the CR can be deleted. Examples
 *   of finalizers include performing backups and deleting resources that are not owned by this CR, like a PVC.
 */
func (r *WorkflowReconciler) finalizeWorkflowManifest(ctx context.Context, req ctrl.Request, workflowManifest *wp5v1alpha1.Workflow) error {
	var logger = log.FromContext(ctx)
	logow.Info(pathLOG + "[finalizeWorkflowManifest] finalizeWorkflowManifest()...")
	err := CleanUpExternalResources(logger, req.Namespace, workflowManifest)
	if err != nil {
		//panic(err)
		return err
	}
	logow.Info(pathLOG + "[finalizeWorkflowManifest] Successfully finalized workflowManifest")
	return nil
}

/**
 * UpdateWorkflowStatus
 */
func (r *WorkflowReconciler) UpdateWorkflowStatus(ctx context.Context, workflowManifest *wp5v1alpha1.Workflow) error {
	logow.Info(pathLOG + "[UpdateWorkflowStatus] Updating Workflow Status ...")

	var condition metav1.Condition
	var conditions []metav1.Condition

	// Update workflow status
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
		logow.Error(pathLOG+"[UpdateWorkflowStatus] Failed to update Workflow status: ", err)
		return err
	}
	logow.Info(pathLOG + "[UpdateWorkflowStatus] UpdateWorkflowStatus() end.")
	return nil
}

/**
 * UpdateActionStatus
 */
func UpdateActionStatus(workflowManifest *wp5v1alpha1.Workflow, action *wp5v1alpha1.Action, message string) {
	logow.Info(pathLOG + "[UpdateActionStatus] Updating Action Status ...")

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
		actionStatus.ActionHost = lookupEnv("PHYSICS_OW_ENDPOINT", "NOT_DEFINED")
		actionStatus.ActionNamespace = namespace + "/" + workflowManifest.Name
		actionStatus.ActionCredentials = lookupEnv("PHYSICS_OW_AUTH", "NOT_DEFINED")
	} else { // new or incremental update
		actionStatus = wp5v1alpha1.ActionStatus{
			Name:              action.Name,
			Namespace:         namespace + "/" + workflowManifest.Name,
			Id:                action.Id,
			Version:           action.Version,
			Runtime:           action.Runtime,
			State:             "Unknown",
			Message:           "",
			BackendURL:        "",
			Remote:            "",
			ActionHost:        lookupEnv("PHYSICS_OW_ENDPOINT", "NOT_DEFINED"), // BackendURL ??
			ActionNamespace:   namespace + "/" + workflowManifest.Name,         // Namespace ??
			ActionCredentials: lookupEnv("PHYSICS_OW_AUTH", "NOT_DEFINED"),     //
		}
		workflowManifest.Status.ActionStatuses = append(workflowManifest.Status.ActionStatuses, actionStatus)
		idx = len(workflowManifest.Status.ActionStatuses) - 1
	}

	var actionError bool = false
	if len(message) > 0 {
		actionError = true
		logow.Warn(pathLOG + "[UpdateActionStatus] actionError: Message: " + message)
	}

	if !actionError {
		logow.Debug(pathLOG + "[UpdateActionStatus] NO 'actionError'. Registering function...")

		actionStatus.State = "Applied" // Registered
		actionStatus.Message = ""
		if _, ok := action.Annotations["remote"]; ok { // Remote function
			actionStatus.Remote = "true"
			actionStatus.BackendURL = "" // For action updates to remote

			// TODO
			/*actionStatus.ActionHost = ""        // BackendURL ??
			actionStatus.ActionNamespace = ""   // Namespace ??
			actionStatus.ActionCredentials = "" //*/

			if input, ok := action.FunctionInput[PHYSICS_ACTION_PROXY_PARAM]; ok {
				if len(input.Value) > 0 {
					actionStatus.State = "Available" // Ready to serving
					actionStatus.BackendURL = input.Value

					// TODO
					/*actionStatus.ActionHost = ""        // BackendURL ??
					actionStatus.ActionNamespace = ""   // Namespace ??
					actionStatus.ActionCredentials = "" //*/
				}
			} else {
				logow.Warn(pathLOG + "[UpdateActionStatus] action.FunctionInput[PHYSICS_ACTION_PROXY_PARAM] is FALSE, with PHYSICS_ACTION_PROXY_PARAM = " + PHYSICS_ACTION_PROXY_PARAM)
			}
		} else { // Local function
			logow.Debug(pathLOG + "[UpdateActionStatus] Local function")

			actionStatus.State = "Available" // Ready to serving
			actionStatus.BackendURL = setBackendURL(&actionStatus, workflowManifest.Annotations["cluster"], action.Annotations["api-id"])

			// TODO
			/*actionStatus.ActionHost = ""        // BackendURL ??
			actionStatus.ActionNamespace = ""   // Namespace ??
			actionStatus.ActionCredentials = "" //*/
		}
	} else {
		logow.Error(pathLOG + "[UpdateActionStatus] ERROR: " + message)
		actionStatus.State = "Error"
		actionStatus.Message = message
	}
	workflowManifest.Status.ActionStatuses[idx] = actionStatus

	logow.Info(pathLOG + "[UpdateActionStatus] Action Status updated")
}

/**
 *
 */
func setBackendURL(actionStatus *wp5v1alpha1.ActionStatus, cluster string, apiID string) string {
	logow.Debug(pathLOG + "[setBackendURL] Setting Backend URL ...")

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

	logow.Debug(pathLOG + "[setBackendURL] result=[" + result + "]")
	return result
}
