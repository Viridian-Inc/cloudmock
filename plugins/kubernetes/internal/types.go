package internal

import (
	"time"
)

// TypeMeta describes an individual object in an API response.
type TypeMeta struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
}

// ObjectMeta is metadata that all persisted resources must have.
type ObjectMeta struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	UID               string            `json:"uid,omitempty"`
	ResourceVersion   string            `json:"resourceVersion,omitempty"`
	CreationTimestamp string            `json:"creationTimestamp,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
}

// ListMeta describes metadata that synthetic resources must have.
type ListMeta struct {
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

// Status is a return value for calls that don't return other objects.
type Status struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata,omitempty"`
	Status   string `json:"status,omitempty"`
	Message  string `json:"message,omitempty"`
	Reason   string `json:"reason,omitempty"`
	Code     int    `json:"code,omitempty"`
}

// Namespace represents a k8s namespace.
type Namespace struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata"`
	Spec       NamespaceSpec   `json:"spec,omitempty"`
	Status     NamespaceStatus `json:"status,omitempty"`
}

type NamespaceSpec struct {
	Finalizers []string `json:"finalizers,omitempty"`
}

type NamespaceStatus struct {
	Phase string `json:"phase,omitempty"`
}

// NamespaceList is a list of Namespaces.
type NamespaceList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata"`
	Items    []Namespace `json:"items"`
}

// Pod represents a k8s pod.
type Pod struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata"`
	Spec       PodSpec   `json:"spec,omitempty"`
	Status     PodStatus `json:"status,omitempty"`
}

type PodSpec struct {
	Containers    []Container `json:"containers,omitempty"`
	NodeName      string      `json:"nodeName,omitempty"`
	RestartPolicy string      `json:"restartPolicy,omitempty"`
}

type Container struct {
	Name    string          `json:"name"`
	Image   string          `json:"image"`
	Ports   []ContainerPort `json:"ports,omitempty"`
	Env     []EnvVar        `json:"env,omitempty"`
	Command []string        `json:"command,omitempty"`
	Args    []string        `json:"args,omitempty"`
}

type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type PodStatus struct {
	Phase      string          `json:"phase,omitempty"`
	PodIP      string          `json:"podIP,omitempty"`
	HostIP     string          `json:"hostIP,omitempty"`
	StartTime  *string         `json:"startTime,omitempty"`
	Conditions []PodCondition  `json:"conditions,omitempty"`
}

type PodCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

// PodList is a list of Pods.
type PodList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata"`
	Items    []Pod `json:"items"`
}

// Service represents a k8s service.
type Service struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata"`
	Spec       ServiceSpec   `json:"spec,omitempty"`
	Status     ServiceStatus `json:"status,omitempty"`
}

type ServiceSpec struct {
	Type      string            `json:"type,omitempty"`
	ClusterIP string            `json:"clusterIP,omitempty"`
	Selector  map[string]string `json:"selector,omitempty"`
	Ports     []ServicePort     `json:"ports,omitempty"`
}

type ServicePort struct {
	Name       string `json:"name,omitempty"`
	Port       int    `json:"port"`
	TargetPort int    `json:"targetPort,omitempty"`
	Protocol   string `json:"protocol,omitempty"`
}

type ServiceStatus struct {
	LoadBalancer LoadBalancerStatus `json:"loadBalancer,omitempty"`
}

type LoadBalancerStatus struct {
	Ingress []LoadBalancerIngress `json:"ingress,omitempty"`
}

type LoadBalancerIngress struct {
	IP       string `json:"ip,omitempty"`
	Hostname string `json:"hostname,omitempty"`
}

// ServiceList is a list of Services.
type ServiceList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata"`
	Items    []Service `json:"items"`
}

// ConfigMap holds configuration data.
type ConfigMap struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata"`
	Data       map[string]string `json:"data,omitempty"`
	BinaryData map[string][]byte `json:"binaryData,omitempty"`
}

// ConfigMapList is a list of ConfigMaps.
type ConfigMapList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata"`
	Items    []ConfigMap `json:"items"`
}

// Secret holds secret data.
type Secret struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata"`
	Data       map[string][]byte `json:"data,omitempty"`
	StringData map[string]string `json:"stringData,omitempty"`
	Type       string            `json:"type,omitempty"`
}

// SecretList is a list of Secrets.
type SecretList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata"`
	Items    []Secret `json:"items"`
}

// Deployment represents a k8s deployment.
type Deployment struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata"`
	Spec       DeploymentSpec   `json:"spec,omitempty"`
	Status     DeploymentStatus `json:"status,omitempty"`
}

type DeploymentSpec struct {
	Replicas int             `json:"replicas,omitempty"`
	Selector *LabelSelector  `json:"selector,omitempty"`
	Template PodTemplateSpec `json:"template,omitempty"`
}

type LabelSelector struct {
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type PodTemplateSpec struct {
	ObjectMeta `json:"metadata,omitempty"`
	Spec       PodSpec `json:"spec,omitempty"`
}

type DeploymentStatus struct {
	Replicas          int `json:"replicas,omitempty"`
	ReadyReplicas     int `json:"readyReplicas,omitempty"`
	AvailableReplicas int `json:"availableReplicas,omitempty"`
	UpdatedReplicas   int `json:"updatedReplicas,omitempty"`
}

// DeploymentList is a list of Deployments.
type DeploymentList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata"`
	Items    []Deployment `json:"items"`
}

// Node represents a k8s node.
type Node struct {
	TypeMeta   `json:",inline"`
	ObjectMeta `json:"metadata"`
	Spec       NodeSpec   `json:"spec,omitempty"`
	Status     NodeStatus `json:"status,omitempty"`
}

type NodeSpec struct {
	PodCIDR    string `json:"podCIDR,omitempty"`
	ProviderID string `json:"providerID,omitempty"`
}

type NodeStatus struct {
	Conditions []NodeCondition `json:"conditions,omitempty"`
	Addresses  []NodeAddress   `json:"addresses,omitempty"`
}

type NodeCondition struct {
	Type   string `json:"type"`
	Status string `json:"status"`
}

type NodeAddress struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

// NodeList is a list of Nodes.
type NodeList struct {
	TypeMeta `json:",inline"`
	ListMeta `json:"metadata"`
	Items    []Node `json:"items"`
}

// WatchEvent is a single event in a watch stream.
type WatchEvent struct {
	Type   string      `json:"type"` // ADDED, MODIFIED, DELETED
	Object interface{} `json:"object"`
}

// Now returns a k8s-formatted timestamp string.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
