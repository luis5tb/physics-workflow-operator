//
// Copyright 2021 Atos
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     https://www.apache.org/licenses/LICENSE-2.0
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
//
package controllers

import (
	"bytes"
	"encoding/json"
	goerrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/go-logr/logr"
	wp5v1alpha1 "gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/api/v1alpha1"
)

var OW_API_FUNC_PATH string = "/api/v1/function"
var OW_API_PACK_PATH string = "/api/v1/package"

//var OW_API_HOST string = "http://localhost:8090"
//var OW_API_HOST string = lookupEnv("PHYSICS_OW_PROXY_ENDPOINT", "http://localhost:8090")
var OW_API_KEY string
var OW_DEFAULT_NAMESPACE string = "_" // "guest" or "_"
var OW_DEFAULT_PACKAGE string = ""    // ""
var OW_DEFAULT_RUNTIME string = "nodejs:default"

type FaaSManager struct {
	Name string
}

func (fm *FaaSManager) CreateFunction(namespace string, action *wp5v1alpha1.Action) (int, string) {
	statusCode, status := fm.UpdateFunction(namespace, action, true)
	return statusCode, status
	//return resp.StatusCode, resp.Status
}

func (fm *FaaSManager) DeleteFunction(namespace string, name string) (int, string) {
	var OW_API_HOST string = lookupEnv("PHYSICS_OW_PROXY_ENDPOINT", "http://localhost:8090")
	var baseUrl string = OW_API_HOST + OW_API_FUNC_PATH
	if len(namespace) == 0 {
		namespace = OW_DEFAULT_NAMESPACE
	}
	url := baseUrl + "/" + name + "?namespace=" + normalizeNamespace(namespace)
	//req, err := http.NewRequest("DELETE", url, bytes.NewBuffer([]byte(name)))
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		// handle error
		//panic(err)
		return http.StatusInternalServerError, err.Error()
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		// handle error
		//panic(err)
		if resp == nil {
			return http.StatusInternalServerError, err.Error()
		}
		return resp.StatusCode, resp.Status
	}
	return resp.StatusCode, resp.Status
}

func (fm *FaaSManager) ReadFunction(namespace string, name string) (int, string) {
	var OW_API_HOST string = lookupEnv("PHYSICS_OW_PROXY_ENDPOINT", "http://localhost:8090")
	var baseUrl string = OW_API_HOST + OW_API_FUNC_PATH
	if len(namespace) == 0 {
		namespace = OW_DEFAULT_NAMESPACE
	}
	url := baseUrl + "/" + name + "?namespace=" + normalizeNamespace(namespace)
	resp, err := http.Get(url)
	if err != nil {
		// handle error
		//panic(err)
		if resp == nil {
			return http.StatusInternalServerError, err.Error()
		}
		return resp.StatusCode, resp.Status
	}
	defer resp.Body.Close()
	//body, err := io.ReadAll(resp.Body)
	if err != nil {
		//panic(err)
		return http.StatusInternalServerError, err.Error()
	}
	//fmt.Println(string(body))
	return resp.StatusCode, resp.Status
}

func (fm *FaaSManager) UpdateFunction(namespace string, action *wp5v1alpha1.Action, isNew bool) (int, string) {
	var OW_API_HOST string = lookupEnv("PHYSICS_OW_PROXY_ENDPOINT", "http://localhost:8090")
	var baseUrl string = OW_API_HOST + OW_API_FUNC_PATH
	var jsonStr []byte
	if len(namespace) == 0 {
		namespace = OW_DEFAULT_NAMESPACE
	}
	//type Annotations
	params := struct {
		Name        string `json:"name"`
		Namespace   string `json:"namespace,omitempty"`
		Version     string `json:"version,omitempty"`
		Runtime     string `json:"runtime,omitempty"`
		Code        string `json:"code,omitempty"`
		Function    string `json:"function,omitempty"`
		Image       string `json:"image,omitempty"`
		Native      bool   `json:"native,omitempty"`
		Web         bool   `json:"web,omitempty"`
		Gateway     bool   `json:"gateway,omitempty"`
		Annotations []struct {
			Key   string `json:"key,omitempty"`
			Value string `json:"value,omitempty"`
		} `json:"annotations,omitempty"`
		Inputs []struct {
			Name  string `json:"name,omitempty"`
			Type  string `json:"type,omitempty"`
			Value string `json:"value,omitempty"`
		} `json:"inputs,omitempty"`
		Limits map[string]int `json:"limits,omitempty"`
		//Limits struct {
		//	Timeout     int `json:"timeout,omitempty"`
		//	Memory      int `json:"memory,omitempty"`
		//	Logs        int `json:"logs,omitempty"`
		//	Concurrency int `json:"concurrency,omitempty"`
		//} `json:"limits,omitempty"`
		Sequence []string `json:"sequence,omitempty"`
	}{
		action.Name, namespace, action.Version, action.Runtime, action.Code,
		"", action.Image, false, true, true, // Action Web interface by default
		nil, nil, nil, nil,
	}
	if len(action.FunctionInput) > 0 {
		params.Inputs = make([]struct {
			Name  string `json:"name,omitempty"`
			Type  string `json:"type,omitempty"`
			Value string `json:"value,omitempty"`
		}, 0)
		for name, value := range action.FunctionInput {
			input := struct {
				Name  string `json:"name,omitempty"`
				Type  string `json:"type,omitempty"`
				Value string `json:"value,omitempty"`
			}{name, value.Type, value.Value}
			params.Inputs = append(params.Inputs, input)
		}
	}
	if len(action.Annotations) > 0 {
		params.Annotations = make([]struct {
			Key   string `json:"key,omitempty"`
			Value string `json:"value,omitempty"`
		}, 0)
		for key, value := range action.Annotations {
			annotation := struct {
				Key   string `json:"key,omitempty"`
				Value string `json:"value,omitempty"`
			}{key, value}
			params.Annotations = append(params.Annotations, annotation)
		}
	}
	//if len(action.Limits) > 0 {
	if len(action.Resources.Limits) > 0 {
		params.Limits = make(map[string]int)
		if _, ok := action.Resources.Limits["memory"]; ok { // OW memory limit in MB <-> K8S memory limit in bytes (string)
			valors := action.Resources.Limits["memory"]
			if valors.Value() > 0 {
				params.Limits["memory"] = int(valors.Value()) // / 1024 / 1024)
			}
		}
		if _, ok := action.Resources.Limits["cpu"]; ok { // OW cpu limit in cores? <-> K8S cpu limit in cores (string)
			valors := action.Resources.Limits["cpu"]
			if valors.Value() > 0 {
				params.Limits["cpu"] = int(valors.Value())
			}
		}
		if _, ok := action.Resources.Limits["storage"]; ok { // OW logs limit in MB <-> K8S Volume limit in bytes (string)
			valors := action.Resources.Limits["storage"]
			if valors.Value() > 0 {
				params.Limits["logs"] = int(valors.Value()) //  / 1024 / 1024)
			}
		}
		/*
			if _, ok := action.Limits["memorySize"]; ok {
				params.Limits["memory"] = action.Limits["memorySize"]
			}
			if _, ok := action.Limits["timeout"]; ok {
				params.Limits["timeout"] = action.Limits["timeout"]
			}
			if _, ok := action.Limits["logsSize"]; ok {
				params.Limits["logs"] = action.Limits["logsSize"]
			}
			if _, ok := action.Limits["concurrency"]; ok {
				params.Limits["concurrency"] = action.Limits["concurrency"]
			}
		*/
	}
	if action.Runtime == "sequence" {
		//params.Sequence = make([]string,0)
		params.Sequence = strings.Split(action.Code, ",")
		params.Code = ""
	}
	params.Namespace = normalizeNamespace(namespace) // namespace/package
	jsonStr, err := json.Marshal(params)
	if err != nil {
		//panic(err)
		return http.StatusInternalServerError, err.Error()
	}
	// Check if package exists
	//fmt.Println(params)
	method := "PUT"
	url := baseUrl
	if isNew { // Create
		method = "POST"
	} else { // Update
		method = "PUT"
		url = baseUrl + "/" + action.Name
	}
	fmt.Println(method, url, string(jsonStr), params)
	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonStr))
	if err != nil {
		// handle error
		//panic(err)
		return http.StatusInternalServerError, err.Error()
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	//resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonStr))
	if err != nil {
		// handle error
		//panic(err)
		if resp == nil {
			return http.StatusInternalServerError, err.Error()
		}
		return resp.StatusCode, resp.Status
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		//panic(err)
		return http.StatusInternalServerError, err.Error()
	} else {
		setActionApiID(action, &body)
	}
	//fmt.Println(string(body))
	return resp.StatusCode, resp.Status
}

func setActionApiID(action *wp5v1alpha1.Action, body *[]byte) {
	//fmt.Println("DEBUG:", string(body))
	var wiredAction map[string]interface{}
	err := json.Unmarshal(*body, &wiredAction)
	//fmt.Println("DEBUG:", err)
	if err == nil {
		//fmt.Println("DEBUG:", wiredAction)
		wiredAnnotations, ok := wiredAction["annotations"].([]interface{})
		if ok {
			//fmt.Println("DEBUG:", wiredAnnotations)
			for k := range wiredAnnotations {
				wiredAnnotation, ok := wiredAnnotations[k].(map[string]interface{})
				if ok {
					//fmt.Println("DEBUG:", wiredAnnotation)
					key, ok := wiredAnnotation["key"].(string)
					if ok && key == "api-id" {
						//fmt.Println("DEBUG:", key)
						value, ok := wiredAnnotation["value"].(string)
						if ok {
							action.Annotations[key] = value
							//fmt.Println("DEBUG:", key, value)
						}
					}
				}
			}
		}
	}
}

/*
Ref: https://github.com/apache/openwhisk/blob/master/docs/reference.md#fully-qualified-names
The fully qualified name of an entity is /namespaceName[/packageName]/entityName. Notice that / is used to
delimit namespaces, packages, and entities.

If the fully qualified name has three parts: /namespaceName/packageName/entityName, then the namespace can be
entered without a prefixed /; otherwise, namespaces must be prefixed with a /.

For convenience, the namespace can be left off if it is the user's default namespace.

namespace					1	Not valid => package name?
package						1	Default namespace
/							2	Default namespace + no packages
/namespace					2	No packages
/package					2	Not valid => namespace?
namespace/					2
package/					2	Not valid	=> namespace + no packages?
namespace/package			2
/namespace/package			3
namespace/package/			3
/namespace/					3	No packages
/package/					3	Not valid	=> namespace + no packages?
/namespace/package/			4

*/
func normalizeNamespace(namespace string) string {
	var result string
	var pckg string
	if len(namespace) == 0 {
		namespace = "/_"
	}
	a := strings.Split(namespace, "/")
	switch len(a) {
	case 1:
		namespace = "_"
		pckg = a[0]
	case 2:
		if strings.HasPrefix(namespace, "/") {
			namespace = a[1]
			pckg = ""
			if a[1] == "" {
				namespace = "_"
			}
		} else {
			namespace = a[0]
			pckg = a[1]
		}
	case 3:
		if strings.HasPrefix(namespace, "/") {
			namespace = a[1]
			pckg = a[2]
		} else {
			namespace = a[0]
			pckg = a[1]
		}
	case 4:
		namespace = a[1]
		pckg = a[2]
	default:
		namespace = "_"
		pckg = ""
	}
	if namespace == "default" { // Kubernetes namespace => Openwhisk namespace
		namespace = "_"
	}
	result = "/" + namespace
	if len(pckg) > 0 {
		result += "/" + pckg
		//result = namespace + "/" + pckg
	}
	return result
} // normalizeNamespace()

func CleanUpExternalResources(logger logr.Logger, namespace string, workflowManifest *wp5v1alpha1.Workflow) error {
	//var logger = log.FromContext(ctx)
	logger.Info("cleanUp()...")
	fm := &FaaSManager{
		Name: "openwhisk",
	}
	pkgInfo := struct{ Name string }{workflowManifest.Name}
	//for pkgName, pkgInfo := range workflowManifest.Spec.Packages {
	//	pkgInfo.Name = pkgName
	//for actionName, action := range pkgInfo.Actions {
	for _, action := range workflowManifest.Spec.Actions {
		//action.Name = actionName
		logger.Info("Function: " + namespace + "/" + pkgInfo.Name + "/" + action.Name)
		statusCode, status := fm.ReadFunction(namespace+"/"+pkgInfo.Name, action.Name)
		logger.Info("Read Function: ", "statusCode", statusCode, "status", status)
		if statusCode == 200 {
			statusCode, status = fm.DeleteFunction(namespace+"/"+pkgInfo.Name, action.Name)
			logger.Info("Delete Function: ", "statusCode", statusCode, "status", status)
			if statusCode != 200 {
				return goerrors.New("Delete function " + namespace + "/" + pkgInfo.Name + "/" + action.Name + " failed! " + status)
			}
		}
	}
	if workflowManifest.Spec.Execution == "NativeSequence" {
		sequence := struct {
			Name        string
			Annotations wp5v1alpha1.Annotations
			Actions     string
		}{workflowManifest.Name, nil, ""}
		//for sequenceName, sequence := range pkgInfo.Sequences {
		//sequence.Name = sequenceName
		logger.Info("Sequence: " + namespace + "/" + pkgInfo.Name + "/" + sequence.Name)
		statusCode, status := fm.ReadFunction(namespace+"/"+pkgInfo.Name, sequence.Name)
		logger.Info("Read Sequence: ", "statusCode", statusCode, "status", status)
		if statusCode == 200 {
			statusCode, status = fm.DeleteFunction(namespace+"/"+pkgInfo.Name, sequence.Name)
			logger.Info("Delete Sequence: ", "statusCode", statusCode, "status", status)
			if statusCode != 200 {
				return goerrors.New("Delete Sequence " + namespace + "/" + pkgInfo.Name + "/" + sequence.Name + " failed! " + status)
			}
		}
	} // if workflowManifest.Spec.Execution == "NativeSequence"
	//}		// for pkgInfo.Sequences
	//}		// for workflowManifest.Spec.Packages
	logger.Info("cleanUp() end.")
	return nil
} // CleanUpExternalResources()

func UpdateExternalResources(logger logr.Logger, namespace string, workflowManifest *wp5v1alpha1.Workflow) error {
	//var logger = log.FromContext(ctx)
	logger.Info("UpdateExternalResources()...")
	var PHYSICS_ACTION_PROXY_IMAGE string = lookupEnv("PHYSICS_ACTION_PROXY_IMAGE", "action-proxy:v1")
	fm := &FaaSManager{
		Name: "openwhisk",
	}
	pkgInfo := struct{ Name string }{workflowManifest.Name}
	//for pkgName, pkgInfo := range workflowManifest.Spec.Packages {
	//	pkgInfo.Name = pkgName
	//	for actionName, action := range pkgInfo.Actions {
	for _, action := range workflowManifest.Spec.Actions {
		//action.Name = actionName
		if len(action.Annotations) == 0 {
			action.Annotations = make(wp5v1alpha1.Annotations)
		}
		action.Annotations["id"] = action.Id
		//	Setup remote actions
		if cluster, ok := workflowManifest.Annotations["cluster"]; ok {
			if target, ok := action.Annotations["cluster"]; ok {
				if cluster != target {
					action.Annotations["remote"] = "true"
					action.Image = PHYSICS_ACTION_PROXY_IMAGE
					action.Runtime = "blackbox"
				}
			}
		}
		//
		logger.Info("Function: " + namespace + "/" + pkgInfo.Name + "/" + action.Name)
		statusCode, status := fm.ReadFunction(namespace+"/"+pkgInfo.Name, action.Name)
		logger.Info("Read Function: ", "statusCode", statusCode, "status", status)
		if statusCode == 404 { // Not found => New
			statusCode, status = fm.CreateFunction(namespace+"/"+pkgInfo.Name, &action)
			logger.Info("Create Function: ", "statusCode", statusCode, "status", status)
			if statusCode != 200 {
				UpdateActionStatus(workflowManifest, &action, status)
				return goerrors.New("Create function " + namespace + "/" + pkgInfo.Name + "/" + action.Name + " failed! " + status)
			}
		} else {
			if statusCode == 200 { // Found => Update
				statusCode, status = fm.UpdateFunction(namespace+"/"+pkgInfo.Name, &action, false)
				logger.Info("Update Function: ", "statusCode", statusCode, "status", status)
				if statusCode != 200 {
					UpdateActionStatus(workflowManifest, &action, status)
					return goerrors.New("Update function " + namespace + "/" + pkgInfo.Name + "/" + action.Name + " failed! " + status)
				}
			}
		}
		if statusCode == 500 { // Connection refused
			UpdateActionStatus(workflowManifest, &action, status)
			return goerrors.New("Create/Update function " + namespace + "/" + pkgInfo.Name + "/" + action.Name + " failed! " + status)
		}
		UpdateActionStatus(workflowManifest, &action, "")
	}
	if workflowManifest.Spec.Execution == "NativeSequence" {
		var actionList string
		for _, actionId := range workflowManifest.Spec.ListOfActions {
			for _, action := range workflowManifest.Spec.Actions {
				if action.Id == actionId.Id {
					actionId.Id = action.Name
					break
				}
			}
			actionList += actionId.Id + ","
		}
		actionList = strings.TrimSuffix(actionList, ",")
		sequence := struct {
			Name        string
			Annotations wp5v1alpha1.Annotations
			Actions     string
		}{workflowManifest.Name, make(wp5v1alpha1.Annotations), actionList}
		sequence.Annotations["id"] = workflowManifest.Annotations["id"]
		//for sequenceName, sequence := range pkgInfo.Sequences {
		//sequence.Name = sequenceName
		logger.Info("Sequence: " + namespace + "/" + pkgInfo.Name + "/" + sequence.Name)
		statusCode, status := fm.ReadFunction(namespace+"/"+pkgInfo.Name, sequence.Name)
		logger.Info("Read Sequence: ", "statusCode", statusCode, "status", status)
		var sequenceAction wp5v1alpha1.Action
		if statusCode == 404 { // Not found => New
			sequenceAction.Name = sequence.Name
			sequenceAction.Runtime = "sequence"
			sequenceAction.Annotations = sequence.Annotations
			sequenceAction.Id = workflowManifest.Annotations["id"]
			sequenceAction.Version = workflowManifest.Annotations["version"]
			//sequenceAction.Code = sequence.Actions
			sequenceAction.Code = normalizeSeqActionsFQN(sequence.Actions, namespace, pkgInfo.Name)
			statusCode, status = fm.CreateFunction(namespace+"/"+pkgInfo.Name, &sequenceAction)
			logger.Info("Create Sequence: ", "statusCode", statusCode, "status", status)
			if statusCode != 200 {
				UpdateActionStatus(workflowManifest, &sequenceAction, status)
				return goerrors.New("Create Sequence " + namespace + "/" + pkgInfo.Name + "/" + sequence.Name + " failed! " + status)
			}
		} else {
			if statusCode == 200 { // Found => Update
				sequenceAction.Name = sequence.Name
				sequenceAction.Runtime = "sequence"
				sequenceAction.Annotations = sequence.Annotations
				sequenceAction.Id = workflowManifest.Annotations["id"]
				sequenceAction.Version = workflowManifest.Annotations["version"]
				//sequenceAction.Code = sequence.Actions
				sequenceAction.Code = normalizeSeqActionsFQN(sequence.Actions, namespace, pkgInfo.Name)
				statusCode, status = fm.UpdateFunction(namespace+"/"+pkgInfo.Name, &sequenceAction, false)
				logger.Info("Update Sequence: ", "statusCode", statusCode, "status", status)
				if statusCode != 200 {
					UpdateActionStatus(workflowManifest, &sequenceAction, status)
					return goerrors.New("Update Sequence " + namespace + "/" + pkgInfo.Name + "/" + sequence.Name + " failed! " + status)
				}
			}
		}
		if statusCode == 500 { // Connection refused
			UpdateActionStatus(workflowManifest, &sequenceAction, status)
			return goerrors.New("Create/Update Sequence " + namespace + "/" + pkgInfo.Name + "/" + sequence.Name + " failed! " + status)
		}
		UpdateActionStatus(workflowManifest, &sequenceAction, "")
	} // if workflowManifest.Spec.Execution == "NativeSequence"
	//}		// for pkgInfo.Sequences
	//}		// for workflowManifest.Spec.Packages
	logger.Info("UpdateExternalResources() end.")
	return nil
} // UpdateExternalResources()

func normalizeSeqActionsFQN(sequence string, namespace string, pckg string) string {
	var result string
	if len(sequence) == 0 {
		return ""
	}
	if len(namespace) == 0 || namespace == "default" {
		namespace = "/_"
	}
	if len(pckg) > 0 {
		namespace += "/" + pckg
	}
	actions := strings.Split(sequence, ",")
	for k := range actions {
		actionPath := namespace
		actionFQN := strings.Split(actions[k], "/")
		actionName := actionFQN[len(actionFQN)-1]
		if len(actionFQN) > 1 {
			actionPath = ""
			for n := range actionFQN {
				if n == len(actionFQN)-1 {
					break
				}
				actionPath += "/" + actionFQN[n]
			}
		}
		if k != 0 {
			result += ","
		}
		//result += sa[k]
		result += normalizeNamespace(actionPath) + "/" + strings.TrimSpace(actionName)
	}
	return result
} // normalizeSeqActionsFQN()

/*
func CreateServiceForOpenWhiskProxy(workflowManifest *wp5v1alpha1.Workflow) *corev1.Service {
	var PHYSICS_OW_PROXY_NAME string = lookupEnv("PHYSICS_OW_PROXY_NAME", "physics-ow-proxy")
	var PHYSICS_OW_PROXY_ENDPOINT string = lookupEnv("PHYSICS_OW_PROXY_ENDPOINT", "http://localhost:8090")
	//var PHYSICS_OW_PROXY_HOST string
	//var PHYSICS_OW_PROXY_PORT int64
	if strings.Count(PHYSICS_OW_PROXY_ENDPOINT, ":") < 2 {
		PHYSICS_OW_PROXY_ENDPOINT += ":80"
	}
	PHYSICS_OW_PROXY_PORT, err := strconv.ParseInt(strings.Split(PHYSICS_OW_PROXY_ENDPOINT, ":")[2], 10, 0)
	if err != nil {
		PHYSICS_OW_PROXY_PORT = 8090
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PHYSICS_OW_PROXY_NAME,
			Namespace: workflowManifest.Namespace,
			Labels:    map[string]string{"app": PHYSICS_OW_PROXY_NAME, "tier": "api"},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{{
				Name:       PHYSICS_OW_PROXY_NAME + "-tcp",
				Protocol:   "TCP",
				Port:       int32(PHYSICS_OW_PROXY_PORT),
				TargetPort: intstr.IntOrString{IntVal: int32(PHYSICS_OW_PROXY_PORT)},
				NodePort:   0,
			}},
			Selector: map[string]string{"app": PHYSICS_OW_PROXY_NAME, "tier": "api"},
		},
	}
	return svc
} // CreateServiceForOpenWhiskProxy()

func CreateDeploymentForOpenWhiskProxy(workflowManifest *wp5v1alpha1.Workflow) *appsv1.Deployment {
	var PHYSICS_OW_PROXY_NAME string = lookupEnv("PHYSICS_OW_PROXY_NAME", "physics-ow-proxy")
	var PHYSICS_OW_PROXY_ENDPOINT string = lookupEnv("PHYSICS_OW_PROXY_ENDPOINT", "http://localhost:8090")
	var PHYSICS_OW_ENDPOINT string = lookupEnv("PHYSICS_OW_ENDPOINT", "http://localhost:3233")
	var PHYSICS_OW_PROXY_IMAGE string = lookupEnv("PHYSICS_OW_PROXY_IMAGE", "physics-ow-proxy:latest")
	var PHYSICS_OW_PROXY_IPP string = lookupEnv("PHYSICS_OW_PROXY_IPP", "Always")
	//var PHYSICS_OW_PROXY_HOST string
	//var PHYSICS_OW_PROXY_PORT int64
	if strings.Count(PHYSICS_OW_PROXY_ENDPOINT, ":") < 2 {
		PHYSICS_OW_PROXY_ENDPOINT += ":80"
	}
	PHYSICS_OW_PROXY_PORT, err := strconv.ParseInt(strings.Split(PHYSICS_OW_PROXY_ENDPOINT, ":")[2], 10, 0)
	if err != nil {
		PHYSICS_OW_PROXY_PORT = 8090
	}
	var replicas int32 = 1
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PHYSICS_OW_PROXY_NAME,
			Namespace: workflowManifest.Namespace,
			Labels:    map[string]string{"app": PHYSICS_OW_PROXY_NAME, "tier": "api"},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": PHYSICS_OW_PROXY_NAME, "tier": "api"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": PHYSICS_OW_PROXY_NAME, "tier": "api"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: PHYSICS_OW_PROXY_IMAGE,
						//Image: "registry.apps.ocphub.physics-faas.eu/wp4/orchestrator:latest",
						//ImagePullPolicy: "Always",
						ImagePullPolicy: corev1.PullPolicy(PHYSICS_OW_PROXY_IPP),
						Name:            PHYSICS_OW_PROXY_NAME,
						Ports: []corev1.ContainerPort{{
							ContainerPort: int32(PHYSICS_OW_PROXY_PORT),
						}},
						Env: []corev1.EnvVar{{
							Name:  "PHYSICS_OW_PROXY_ENDPOINT",
							Value: PHYSICS_OW_PROXY_ENDPOINT,
						},
							{
								Name:  "PHYSICS_OW_ENDPOINT",
								Value: PHYSICS_OW_ENDPOINT,
							}},
					}},
					//ImagePullSecrets: []corev1.LocalObjectReference{{
					//	Name: "regcred",
					//}},
				},
			},
		},
	}
	// Set OpenwhiskManifest instance as the owner and controller
	//ctrl.SetControllerReference(openwhiskManifest, dep, r.Scheme)
	return dep
} // CreateDeploymentForOpenWhiskProxy()
*/
func lookupEnv(name string, value string) string {
	result, ok := os.LookupEnv(name) // os.Getenv(name)
	if !ok {
		result = value
	}
	return result
} // lookupEnv()
