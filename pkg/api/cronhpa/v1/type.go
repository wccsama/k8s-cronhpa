package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronHPA resource
type CronHPA struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec CronHPASpec `json:"spec,omitempty"`

	Status CronHPAStatus `json:"status,omitempty"`
}

// CronHPASpec describes the attributes that a CronHPA is created with.
type CronHPASpec struct {
	ScaleTargetRef          ScaleTargetRef `json:"scaleTargetRef"`
	Crons                   []Cron         `json:"crons"`
	ExcludeDates            []string       `json:"excludeDates"`
	StartingDeadlineSeconds int32          `json:"startingDeadlineSeconds"`
}

// CronJob defines the action of scaling
type Cron struct {
	Name      string `json:"name"`
	Schedule  string `json:"schedule"`
	RunOnce   bool   `json:"runOnce"`
	TargeSize int32  `json:"targetSize"`
}

// CronHPAStatus is information about the current status of a CronHPA.
type CronHPAStatus struct {
	// Information when was the last time the schedule was successfully scheduled.
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty" protobuf:"bytes,2,opt,name=lastScheduleTime"`

	Conditions []CronHPAConditions `json:"conditions"`
}

// JobStatus status
type JobStatus string

const (
	// Failed job
	Failed JobStatus = "Failed"
	// Submitted job
	Submitted JobStatus = "Submitted"
	// Succeed job
	Succeed JobStatus = "Succeed"
)

// CronHPAConditions contains condition information for a CronHPA
type CronHPAConditions struct {
	Name          string      `json:"name"`
	Schedule      string      `json:"schedule"`
	RunOnce       bool        `json:"runOnce"`
	TargeSize     string      `json:"targetSize"`
	Status        JobStatus   `json:"Status"`
	LastProbeTime metav1.Time `json:"lastProbeTime"`
	Message       string      `json:"message"`
}

// ScaleTargetRef contains enough information to let you identify the referred resource
type ScaleTargetRef struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronHPAList is a set of CronHPA
type CronHPAList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []CronHPA `json:"items"`
}
