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
package v1alpha1

import (
	typesv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// WorkflowSpec defines the desired state of Workflow
type WorkflowSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Type string `json:"type,omitempty"`
	// Platform is the target platform, it can be OpenWhisk, Knative, ...
	Platform string `json:"platform,omitempty"`
	// Exection is the type of execution mode: NativeSequence, NoderedSequence, or Service
	Execution string `json:"execution,omitempty"`
	Native    bool   `json:"native,omitempty"`

	// ListOfActions is the ordered list of actions to execute
	ListOfActions []ActionId `json:"listOfActions,omitempty"`

	Actions []Action `json:"actions"`
}

type ActionId struct {
	Id string `json:"id"`
}

type Action struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Id          string `json:"id"`
	Version     string `json:"version,omitempty"`
	// Runtime is the function runtime: NodeJS, Python, ...
	Runtime string `json:"runtime"`
	// CodeRepo: function code is obtained from a repo
	CodeRepo string `json:"codeRepo,omitempty"`
	// Code: function code is passed as string here
	Code string `json:"code,omitempty"`
	// Image: function code in a docker image
	Image         string      `json:"image,omitempty"`
	FunctionInput Inputs      `json:"functionInput,omitempty"`
	Annotations   Annotations `json:"annotations,omitempty"`
	// Same resources as the core k8s for containers: https://github.com/kubernetes/api/blob/master/core/v1/types.go
	// This includes limits and requests
	Resources typesv1.ResourceRequirements `json:"resources,omitempty"`
	// ExtraResources: Other resources needed by physics, such as cpu or memory
	ExtraResources ExtraResourcesInfo `json:"extraResources,omitempty"`
	// PerformanceProfile: Information provided by the Performance Profiler module
	PerformanceProfile PerformanceProfileInfo `json:"performanceProfile,omitempty"`
	// New
	// Default Parameters
	DefaultParams []DefautlParameters `json:"defaultParams,omitempty"`
}

type DefautlParameters struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

// Input params per function
type Inputs map[string]Input
type Input struct {
	Name        string `json:"name,omitempty"`
	Value       string `json:"value,omitempty"`
	Default     string `json:"default,omitempty"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// Annotations per function
type Annotations map[string]string

type ExtraResourcesInfo struct {
	Gpu      bool   `json:"gpu,omitempty"`
	DiskType string `json:"diskType,omitempty"`
}

type PerformanceProfileInfo struct {
	Cpu                string `json:"cpu,omitempty"`
	Memory             string `json:"memory,omitempty"`
	FsReads            string `json:"fsReads,omitempty"`
	FsWrites           string `json:"fsWrites,omitempty"`
	NetworkReceived    string `json:"networkReceived,omitempty"`
	NetworkTransmitted string `json:"networkTransmitted,omitempty"`
}

// WorkflowStatus defines the observed state of Workflow
type WorkflowStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Conditions represent the latest available observations of an object's state
	Conditions     []metav1.Condition `json:"conditions,omitempty"`
	ActionStatuses []ActionStatus     `json:"actionStatus,omitempty"`
	ActionStatSer  string             `json:"actionStatSer,omitempty"`
}

type ActionStatus struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"` // namespace + package ?
	//Package	string `json:"package"`
	Id         string `json:"id"`
	Version    string `json:"version"`
	Runtime    string `json:"runtime"`
	State      string `json:"state"` // Unknown, Applied, Available, Error
	Message    string `json:"message"`
	BackendURL string `json:"backendURL"`
	Remote     string `json:"remote,omitempty"`
	// New fields
	ActionHost        string `json:"actionHost,omitempty"`
	ActionNamespace   string `json:"actionNamespace,omitempty"`
	ActionCredentials string `json:"actionCredentials,omitempty"`
	ActionParam1      string `json:"actionParam1,omitempty"`
	ActionParam2      string `json:"actionParam2,omitempty"`
}

// Workflow is the Schema for the workflows API
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkflowList contains a list of Workflow
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
