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

	//+kubebuilder:validation:MinLength=0
	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`

	//+kubebuilder:validation:MinLength=0
	// The name of the application to monitor for healthcheck
	// +optional
	Name string `json:"name,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// URL to GET healthcheck data for
	Url string `json:"url"`

	//+kubebuilder:validation:Minimum=1
	//+kubebuilder:validation:Maximum=5
	//+kubebuilder:default:=5
	// The number of retries to publish healthcheck data
	Retries int32 `json:"retries"`

	//+kubebuilder:validation:Minimum=0
	// Optional healthcheck data response code
	// +optional
	Response int `json:"res_code,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// The UUID of the message consumed from the producer
	HTTP_Check_Id string `json:"http_check_id"`

	//+kubebuilder:validation:MinLength=0
	// Optional Kafka Broker 0 ID:PORT
	// +optional
	Broker_0 string `json:"broker_0,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// Optional Kafka Broker 1 ID:PORT
	// +optional
	Broker_1 string `json:"broker_1,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// Optional Kafka Broker 2 ID:PORT
	// +optional
	Broker_2 string `json:"broker_2,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// Optional Kafka Client ID
	// +optional
	Client_Id string `json:"client_id,omitempty"`

	//+kubebuilder:validation:MinLength=0
	// Optional Kafka Topic to connect the producer to the broker and publish the
	// healthcheck message
	// +optional
	Topic string `json:"topic,omitempty"`

	// Optional Base64 encoded Quay.io robot credentials to pull private docker images
	// +optional
	DockerConfigJSON string `json:"dockerConfigJSON,omitempty"`

	//+kubebuilder:default:=0
	// The successful jobs history limit
	SuccessfulJobsHistoryLimit int32 `json:"success_limit,omitempty"`

	//+kubebuilder:default:=0
	// The failed jobs history limit
	FailedJobsHistoryLimit int32 `json:"failure_limit,omitempty"`
}

// CronStatus defines the observed state of Cron
type CronStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Information when was the last time the job was successful.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	//+kubebuilder:default=false
	// Status of the CronJob
	// +optional
	Active bool `json:"active,omitempty"`
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
