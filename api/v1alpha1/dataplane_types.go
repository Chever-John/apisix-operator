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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DataPlaneSpec defines the desired state of DataPlane
type DataPlaneSpec struct {
	DataPlaneDeploymentOptions `json:",inline"`
}

type DataPlaneDeploymentOptions struct {
	DeploymentOptions `json:",inline"`
}

// DataPlaneStatus defines the observed state of DataPlane
type DataPlaneStatus struct {
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	Service string `json:"service,omitempty"`
}

//+genclient
//+k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=kdp,categories=apisix;all
//+kubebuilder:printcolumn:name="Ready",description="The Resource is ready",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].status`
//+kubebuilder:printcolumn:name="Provisioned",description="The Resource is provisioned",type=string,JSONPath=`.status.conditions[?(@.type=='Provisioned')].status`

// DataPlane is the Schema for the dataplanes API
type DataPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DataPlaneSpec   `json:"spec,omitempty"`
	Status DataPlaneStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DataPlaneList contains a list of DataPlane
type DataPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DataPlane `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DataPlane{}, &DataPlaneList{})
}

func (d *DataPlane) GetConditions() []metav1.Condition {
	return d.Status.Conditions

}

func (d *DataPlane) SetConditions(conditions []metav1.Condition) {
	d.Status.Conditions = conditions

}
