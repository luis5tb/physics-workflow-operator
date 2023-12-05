package controllers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"reflect"

	"github.com/go-logr/logr"
	wp5v1alpha1 "gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/api/v1alpha1"
	logow "gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/common/logs"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

/**
 * UpdateKnativeWorkflowStatus
 */
func UpdateKnativeWorkflowStatus(r *WorkflowReconciler, ctx context.Context, workflowManifest *wp5v1alpha1.Workflow, route string) error {
	logow.Info(pathLOG + "[UpdateKnativeWorkflowStatus] Updating Workflow Status ...")
	needs_status_update := false
	if !reflect.DeepEqual("Available", workflowManifest.Status.ActionStatuses[0].State) {
		workflowManifest.Status.ActionStatuses[0].State = "Available"
		needs_status_update = true
	}
	if !reflect.DeepEqual(route, workflowManifest.Status.ActionStatuses[0].BackendURL) {
		workflowManifest.Status.ActionStatuses[0].BackendURL = route
		needs_status_update = true
	}
	if needs_status_update {
		err := r.Status().Update(ctx, workflowManifest)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReconcileKnativeResources(r *WorkflowReconciler, ctx context.Context, logger logr.Logger, namespace string, workflowManifest *wp5v1alpha1.Workflow) (string, error) {
	gvk := schema.GroupVersionKind{
		Group:   "serving.knative.dev",
		Version: "v1",
		Kind:    "Service",
	}

	for _, action := range workflowManifest.Spec.Actions {
		// Get the service
		kn_service := &unstructured.Unstructured{}
		kn_service.SetGroupVersionKind(gvk)
		kn_service.SetName(action.Name)
		kn_service.SetNamespace(workflowManifest.ObjectMeta.Namespace)

		err := r.Client.Get(ctx, client.ObjectKeyFromObject(kn_service), kn_service)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				// if not exists, create
				kn_service, err = generateNewService(r, action)
				if err != nil {
					err = fmt.Errorf("Failed to generate the Knative Service: %v", err)
					return "", err
				}
				kn_service.SetNamespace(workflowManifest.ObjectMeta.Namespace)
				if err := r.Client.Create(ctx, kn_service); err != nil {
					err = fmt.Errorf("Failed to deploy the Knative Service: %v", err)
					return "", err
				}
			} else {
				// error getting the service - requeue the request.
				return "", err
			}
		} else {
			// if exists update
			updated_service, err := updateService(r, action, kn_service)
			if err != nil {
				err = fmt.Errorf("Failed to regenerated the updated Knative Service: %v", err)
				return "", err
			}
			if err := r.Client.Update(ctx, updated_service); err != nil {
				err = fmt.Errorf("Failed to update the Knative Service: %v", err)
				return "", err
			}
		}
	}

	// only one action per workflow allowed at the moment
	// get route
	route, err := getRoute(r, ctx, workflowManifest.Spec.Actions[0].Name, workflowManifest.ObjectMeta.Namespace)
	if err != nil {
		//
		err = fmt.Errorf("Failed to get the Route: %v", err)
		return "", err
	}
	return route, nil
}

func generateNewService(r *WorkflowReconciler, action wp5v1alpha1.Action) (*unstructured.Unstructured, error) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("serving.knative.dev/v1") // Replace with your API version
	obj.SetKind("Service")                      // Replace with your resource kind
	obj.SetName(action.Name)

	concurrency := 1
	annotations := make(map[string]string)
	for _, elem := range action.DefaultParams {
		if elem.Name == "maxscale" {
			annotations["autoscaling.knative.dev/maxScale"] = elem.Value
		}
		if elem.Name == "minscale" {
			annotations["autoscaling.knative.dev/minScale"] = elem.Value
		}
		if elem.Name == "concurrency" {
			// Convert the string to an integer
			num, err := strconv.Atoi(elem.Value)
			// Check for conversion errors
			if err != nil {
				err = fmt.Errorf("Conversion error for concurrencty: %v", err)
				return obj, err
			} else {
				concurrency = num
			}
		}
	}

	// Set spec fields
	spec := make(map[string]interface{})
	template := make(map[string]interface{})

	metadata := make(map[string]interface{})
	metadata["annotations"] = annotations

	template_spec := make(map[string]interface{})
	template_spec["containerConcurrency"] = concurrency

	container := []interface{}{
		map[string]interface{}{
			"image": action.Image,
		},
	}
	template_spec["containers"] = container
	template["metadata"] = metadata
	template["spec"] = template_spec
	spec["template"] = template

	obj.Object["spec"] = spec

	return obj, nil
}

func updateService(r *WorkflowReconciler, action wp5v1alpha1.Action, kn_s *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	concurrency := 1
	// Get the current annotations
	annotations, found, err := unstructured.NestedMap(kn_s.Object, "spec", "template", "metadata", "annotations")
	if err != nil || !found {
		return kn_s, err
	}
	for _, elem := range action.DefaultParams {
		if elem.Name == "maxscale" {
			annotations["autoscaling.knative.dev/maxScale"] = elem.Value
		}
		if elem.Name == "minscale" {
			annotations["autoscaling.knative.dev/minScale"] = elem.Value
		}
		if elem.Name == "concurrency" {
			// Convert the string to an integer
			num, err := strconv.Atoi(elem.Value)
			// Check for conversion errors
			if err != nil {
				err = fmt.Errorf("Conversion error for concurrencty: %v", err)
				return kn_s, err
			} else {
				concurrency = num
			}
		}
	}

	// Update the annotations in the object
	unstructured.SetNestedField(kn_s.Object, annotations, "spec", "template", "metadata", "annotations")

	// Update containerConcurrency
	unstructured.SetNestedField(kn_s.Object, concurrency, "spec", "template", "spec", "containerConcurrency")

	// Update container image
	unstructured.SetNestedField(kn_s.Object, action.Image, "spec", "template", "spec", "containers", "0", "image")

	return kn_s, nil
}

func getRoute(r *WorkflowReconciler, ctx context.Context, knf_name string, knf_namespace string) (string, error) {
	gvk := schema.GroupVersionKind{
		Group:   "serving.knative.dev",
		Version: "v1",
		Kind:    "Service",
	}

	// Start time for timeout
	startTime := time.Now()
	timeout := 120 * time.Second

	for {
		// Check if timeout has been reached
		if time.Since(startTime) > timeout {
			return "", fmt.Errorf("timed out waiting for route")
		}

		// Retrieve the object
		obj := &unstructured.Unstructured{}
		obj.SetGroupVersionKind(gvk)
		obj.SetName(knf_name)
		obj.SetNamespace(knf_namespace)

		err := r.Client.Get(ctx, client.ObjectKeyFromObject(obj), obj)
		if err != nil {
			return "", fmt.Errorf("failed to get Knative Service: %v", err)
		}

		// Extract the status field
		status, found, err := unstructured.NestedString(obj.Object, "status", "url")
		if err != nil || !found || status == "" {
			// If status field is not yet populated, wait for a short duration before checking again
			time.Sleep(5 * time.Second)
			continue
		}

		// Status field is populated, return the URL
		return status, nil
	}
}
