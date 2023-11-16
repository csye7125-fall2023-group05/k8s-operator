/*
Copyright 2023.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CronSpec defines the desired state of Cron
type CronSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Cron. Edit cron_types.go to remove/update
	Url       string `json:"url,omitempty"`
	Retries   int    `json:"retries,omitempty"`
	Response  int    `json:"res_code,omitempty"`
	Broker_0  string `json:"broker_0,omitempty"`
	Broker_1  string `json:"broker_1,omitempty"`
	Broker_2  string `json:"broker_2,omitempty"`
	Client_Id string `json:"client_id,omitempty"`
	Topic     string `json:"topic,omitempty"`
}

// CronStatus defines the observed state of Cron
type CronStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastSuccessfulTime *metav1.Time `json:"lastSuccessfulTime,omitempty" protobuf:"bytes,5,opt,name=lastSuccessfulTime"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Cron is the Schema for the crons API
type Cron struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronSpec   `json:"spec,omitempty"`
	Status CronStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// CronList contains a list of Cron
type CronList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cron `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cron{}, &CronList{})
}
