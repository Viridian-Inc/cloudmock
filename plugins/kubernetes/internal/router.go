package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Router maps Kubernetes API paths to the appropriate handlers.
type Router struct {
	store *Store
}

// NewRouter creates a k8s API router backed by the given store.
func NewRouter(store *Store) *Router {
	return &Router{store: store}
}

// Handle processes a k8s API request and returns (statusCode, responseBody).
func (rt *Router) Handle(method, path string, body []byte, queryParams map[string]string) (int, []byte) {
	// Strip trailing slash.
	path = strings.TrimRight(path, "/")

	// --- Discovery endpoints ---
	if path == "/api" {
		return rt.handleAPIVersions()
	}
	if path == "/api/v1" {
		return rt.handleCoreV1Resources()
	}
	if path == "/apis" {
		return rt.handleAPIGroupList()
	}
	if path == "/apis/apps/v1" || path == "/apis/apps" {
		return rt.handleAppsV1Resources()
	}
	if path == "/version" {
		return jsonResponse(200, map[string]string{
			"major": "1", "minor": "28",
			"gitVersion": "v1.28.0-cloudmock",
			"platform":   "cloudmock/amd64",
		})
	}

	// --- Core v1 resources: /api/v1/... ---
	if strings.HasPrefix(path, "/api/v1") {
		return rt.handleCoreV1(method, path, body, queryParams)
	}

	// --- Apps v1 resources: /apis/apps/v1/... ---
	if strings.HasPrefix(path, "/apis/apps/v1") {
		return rt.handleAppsV1(method, path, body, queryParams)
	}

	return jsonResponse(404, Status{
		TypeMeta: TypeMeta{Kind: "Status", APIVersion: "v1"},
		Status:   "Failure",
		Message:  fmt.Sprintf("the server could not find the requested resource: %s", path),
		Reason:   "NotFound",
		Code:     404,
	})
}

// --- Discovery ---

func (rt *Router) handleAPIVersions() (int, []byte) {
	return jsonResponse(200, map[string]interface{}{
		"kind":     "APIVersions",
		"versions": []string{"v1"},
		"serverAddressByClientCIDRs": []map[string]string{
			{"clientCIDR": "0.0.0.0/0", "serverAddress": "localhost:4566"},
		},
	})
}

func (rt *Router) handleCoreV1Resources() (int, []byte) {
	resources := []map[string]interface{}{
		{"name": "namespaces", "singularName": "namespace", "namespaced": false, "kind": "Namespace", "verbs": []string{"create", "delete", "get", "list", "watch"}},
		{"name": "pods", "singularName": "pod", "namespaced": true, "kind": "Pod", "verbs": []string{"create", "delete", "get", "list", "watch"}},
		{"name": "services", "singularName": "service", "namespaced": true, "kind": "Service", "verbs": []string{"create", "delete", "get", "list"}},
		{"name": "configmaps", "singularName": "configmap", "namespaced": true, "kind": "ConfigMap", "verbs": []string{"create", "delete", "get", "list"}},
		{"name": "secrets", "singularName": "secret", "namespaced": true, "kind": "Secret", "verbs": []string{"create", "delete", "get", "list"}},
		{"name": "nodes", "singularName": "node", "namespaced": false, "kind": "Node", "verbs": []string{"get", "list"}},
		{"name": "serviceaccounts", "singularName": "serviceaccount", "namespaced": true, "kind": "ServiceAccount", "verbs": []string{"get", "list"}},
	}
	return jsonResponse(200, map[string]interface{}{
		"kind":         "APIResourceList",
		"groupVersion": "v1",
		"resources":    resources,
	})
}

func (rt *Router) handleAPIGroupList() (int, []byte) {
	return jsonResponse(200, map[string]interface{}{
		"kind":       "APIGroupList",
		"apiVersion": "v1",
		"groups": []map[string]interface{}{
			{
				"name": "apps",
				"versions": []map[string]string{
					{"groupVersion": "apps/v1", "version": "v1"},
				},
				"preferredVersion": map[string]string{"groupVersion": "apps/v1", "version": "v1"},
			},
			{
				"name": "batch",
				"versions": []map[string]string{
					{"groupVersion": "batch/v1", "version": "v1"},
				},
				"preferredVersion": map[string]string{"groupVersion": "batch/v1", "version": "v1"},
			},
			{
				"name": "networking.k8s.io",
				"versions": []map[string]string{
					{"groupVersion": "networking.k8s.io/v1", "version": "v1"},
				},
				"preferredVersion": map[string]string{"groupVersion": "networking.k8s.io/v1", "version": "v1"},
			},
		},
	})
}

func (rt *Router) handleAppsV1Resources() (int, []byte) {
	resources := []map[string]interface{}{
		{"name": "deployments", "singularName": "deployment", "namespaced": true, "kind": "Deployment", "verbs": []string{"create", "delete", "get", "list", "watch"}},
		{"name": "replicasets", "singularName": "replicaset", "namespaced": true, "kind": "ReplicaSet", "verbs": []string{"get", "list"}},
		{"name": "statefulsets", "singularName": "statefulset", "namespaced": true, "kind": "StatefulSet", "verbs": []string{"get", "list"}},
		{"name": "daemonsets", "singularName": "daemonset", "namespaced": true, "kind": "DaemonSet", "verbs": []string{"get", "list"}},
	}
	return jsonResponse(200, map[string]interface{}{
		"kind":         "APIResourceList",
		"groupVersion": "apps/v1",
		"resources":    resources,
	})
}

// --- Core v1 handlers ---

func (rt *Router) handleCoreV1(method, path string, body []byte, queryParams map[string]string) (int, []byte) {
	// Remove prefix: /api/v1
	rest := strings.TrimPrefix(path, "/api/v1")
	if rest == "" {
		return rt.handleCoreV1Resources()
	}

	parts := splitPath(rest)
	// /api/v1/namespaces
	if len(parts) == 1 && parts[0] == "namespaces" {
		switch method {
		case http.MethodGet:
			items := rt.store.ListNamespaces()
			return jsonResponse(200, NamespaceList{
				TypeMeta: TypeMeta{Kind: "NamespaceList", APIVersion: "v1"},
				Items:    items,
			})
		case http.MethodPost:
			var ns Namespace
			if err := json.Unmarshal(body, &ns); err != nil {
				return statusError(400, "BadRequest", err.Error())
			}
			ns.TypeMeta = TypeMeta{Kind: "Namespace", APIVersion: "v1"}
			if err := rt.store.CreateNamespace(&ns); err != nil {
				return statusError(409, "AlreadyExists", err.Error())
			}
			return jsonResponse(201, ns)
		}
	}

	// /api/v1/namespaces/{name}
	if len(parts) == 2 && parts[0] == "namespaces" {
		name := parts[1]
		switch method {
		case http.MethodGet:
			ns, ok := rt.store.GetNamespace(name)
			if !ok {
				return statusError(404, "NotFound", fmt.Sprintf("namespaces %q not found", name))
			}
			return jsonResponse(200, ns)
		case http.MethodDelete:
			if err := rt.store.DeleteNamespace(name); err != nil {
				return statusError(404, "NotFound", err.Error())
			}
			return jsonResponse(200, Status{TypeMeta: TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: "Success", Code: 200})
		}
	}

	// /api/v1/nodes
	if len(parts) == 1 && parts[0] == "nodes" && method == http.MethodGet {
		items := rt.store.ListNodes()
		return jsonResponse(200, NodeList{
			TypeMeta: TypeMeta{Kind: "NodeList", APIVersion: "v1"},
			Items:    items,
		})
	}

	// /api/v1/nodes/{name}
	if len(parts) == 2 && parts[0] == "nodes" && method == http.MethodGet {
		node, ok := rt.store.GetNode(parts[1])
		if !ok {
			return statusError(404, "NotFound", fmt.Sprintf("nodes %q not found", parts[1]))
		}
		return jsonResponse(200, node)
	}

	// /api/v1/pods (all namespaces)
	if len(parts) == 1 && parts[0] == "pods" && method == http.MethodGet {
		items := rt.store.ListAllPods()
		return jsonResponse(200, PodList{
			TypeMeta: TypeMeta{Kind: "PodList", APIVersion: "v1"},
			Items:    items,
		})
	}

	// /api/v1/namespaces/{ns}/{resource} and /api/v1/namespaces/{ns}/{resource}/{name}
	if len(parts) >= 3 && parts[0] == "namespaces" {
		ns := parts[1]
		resource := parts[2]
		var resourceName string
		if len(parts) >= 4 {
			resourceName = parts[3]
		}
		return rt.handleNamespacedResource(method, ns, resource, resourceName, body, queryParams)
	}

	return statusError(404, "NotFound", fmt.Sprintf("path not found: %s", path))
}

func (rt *Router) handleNamespacedResource(method, ns, resource, name string, body []byte, queryParams map[string]string) (int, []byte) {
	switch resource {
	case "pods":
		return rt.handlePods(method, ns, name, body, queryParams)
	case "services":
		return rt.handleServices(method, ns, name, body)
	case "configmaps":
		return rt.handleConfigMaps(method, ns, name, body)
	case "secrets":
		return rt.handleSecrets(method, ns, name, body)
	default:
		return statusError(404, "NotFound", fmt.Sprintf("resource %q not found", resource))
	}
}

// --- Pods ---

func (rt *Router) handlePods(method, ns, name string, body []byte, queryParams map[string]string) (int, []byte) {
	if name == "" {
		switch method {
		case http.MethodGet:
			labelSelector := queryParams["labelSelector"]
			var items []Pod
			if labelSelector != "" {
				items = rt.store.FilterPodsByLabels(ns, labelSelector)
			} else {
				items = rt.store.ListPods(ns)
			}
			return jsonResponse(200, PodList{
				TypeMeta: TypeMeta{Kind: "PodList", APIVersion: "v1"},
				Items:    items,
			})
		case http.MethodPost:
			var pod Pod
			if err := json.Unmarshal(body, &pod); err != nil {
				return statusError(400, "BadRequest", err.Error())
			}
			pod.TypeMeta = TypeMeta{Kind: "Pod", APIVersion: "v1"}
			pod.Namespace = ns
			if err := rt.store.CreatePod(&pod); err != nil {
				return statusError(409, "AlreadyExists", err.Error())
			}
			return jsonResponse(201, pod)
		}
	} else {
		switch method {
		case http.MethodGet:
			pod, ok := rt.store.GetPod(ns, name)
			if !ok {
				return statusError(404, "NotFound", fmt.Sprintf("pods %q not found", name))
			}
			return jsonResponse(200, pod)
		case http.MethodDelete:
			if err := rt.store.DeletePod(ns, name); err != nil {
				return statusError(404, "NotFound", err.Error())
			}
			return jsonResponse(200, Status{TypeMeta: TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: "Success", Code: 200})
		}
	}
	return statusError(405, "MethodNotAllowed", "method not allowed")
}

// --- Services ---

func (rt *Router) handleServices(method, ns, name string, body []byte) (int, []byte) {
	if name == "" {
		switch method {
		case http.MethodGet:
			items := rt.store.ListServices(ns)
			return jsonResponse(200, ServiceList{
				TypeMeta: TypeMeta{Kind: "ServiceList", APIVersion: "v1"},
				Items:    items,
			})
		case http.MethodPost:
			var svc Service
			if err := json.Unmarshal(body, &svc); err != nil {
				return statusError(400, "BadRequest", err.Error())
			}
			svc.TypeMeta = TypeMeta{Kind: "Service", APIVersion: "v1"}
			svc.Namespace = ns
			if err := rt.store.CreateService(&svc); err != nil {
				return statusError(409, "AlreadyExists", err.Error())
			}
			return jsonResponse(201, svc)
		}
	} else {
		switch method {
		case http.MethodGet:
			svc, ok := rt.store.GetService(ns, name)
			if !ok {
				return statusError(404, "NotFound", fmt.Sprintf("services %q not found", name))
			}
			return jsonResponse(200, svc)
		case http.MethodDelete:
			if err := rt.store.DeleteService(ns, name); err != nil {
				return statusError(404, "NotFound", err.Error())
			}
			return jsonResponse(200, Status{TypeMeta: TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: "Success", Code: 200})
		}
	}
	return statusError(405, "MethodNotAllowed", "method not allowed")
}

// --- ConfigMaps ---

func (rt *Router) handleConfigMaps(method, ns, name string, body []byte) (int, []byte) {
	if name == "" {
		switch method {
		case http.MethodGet:
			items := rt.store.ListConfigMaps(ns)
			return jsonResponse(200, ConfigMapList{
				TypeMeta: TypeMeta{Kind: "ConfigMapList", APIVersion: "v1"},
				Items:    items,
			})
		case http.MethodPost:
			var cm ConfigMap
			if err := json.Unmarshal(body, &cm); err != nil {
				return statusError(400, "BadRequest", err.Error())
			}
			cm.TypeMeta = TypeMeta{Kind: "ConfigMap", APIVersion: "v1"}
			cm.Namespace = ns
			if err := rt.store.CreateConfigMap(&cm); err != nil {
				return statusError(409, "AlreadyExists", err.Error())
			}
			return jsonResponse(201, cm)
		}
	} else {
		switch method {
		case http.MethodGet:
			cm, ok := rt.store.GetConfigMap(ns, name)
			if !ok {
				return statusError(404, "NotFound", fmt.Sprintf("configmaps %q not found", name))
			}
			return jsonResponse(200, cm)
		case http.MethodDelete:
			if err := rt.store.DeleteConfigMap(ns, name); err != nil {
				return statusError(404, "NotFound", err.Error())
			}
			return jsonResponse(200, Status{TypeMeta: TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: "Success", Code: 200})
		}
	}
	return statusError(405, "MethodNotAllowed", "method not allowed")
}

// --- Secrets ---

func (rt *Router) handleSecrets(method, ns, name string, body []byte) (int, []byte) {
	if name == "" {
		switch method {
		case http.MethodGet:
			items := rt.store.ListSecrets(ns)
			return jsonResponse(200, SecretList{
				TypeMeta: TypeMeta{Kind: "SecretList", APIVersion: "v1"},
				Items:    items,
			})
		case http.MethodPost:
			var sec Secret
			if err := json.Unmarshal(body, &sec); err != nil {
				return statusError(400, "BadRequest", err.Error())
			}
			sec.TypeMeta = TypeMeta{Kind: "Secret", APIVersion: "v1"}
			sec.Namespace = ns
			if err := rt.store.CreateSecret(&sec); err != nil {
				return statusError(409, "AlreadyExists", err.Error())
			}
			return jsonResponse(201, sec)
		}
	} else {
		switch method {
		case http.MethodGet:
			sec, ok := rt.store.GetSecret(ns, name)
			if !ok {
				return statusError(404, "NotFound", fmt.Sprintf("secrets %q not found", name))
			}
			return jsonResponse(200, sec)
		case http.MethodDelete:
			if err := rt.store.DeleteSecret(ns, name); err != nil {
				return statusError(404, "NotFound", err.Error())
			}
			return jsonResponse(200, Status{TypeMeta: TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: "Success", Code: 200})
		}
	}
	return statusError(405, "MethodNotAllowed", "method not allowed")
}

// --- Apps v1 handlers ---

func (rt *Router) handleAppsV1(method, path string, body []byte, queryParams map[string]string) (int, []byte) {
	rest := strings.TrimPrefix(path, "/apis/apps/v1")
	if rest == "" {
		return rt.handleAppsV1Resources()
	}

	parts := splitPath(rest)

	// /apis/apps/v1/namespaces/{ns}/deployments
	if len(parts) >= 3 && parts[0] == "namespaces" {
		ns := parts[1]
		resource := parts[2]
		var name string
		if len(parts) >= 4 {
			name = parts[3]
		}
		if resource == "deployments" {
			return rt.handleDeployments(method, ns, name, body)
		}
	}

	return statusError(404, "NotFound", fmt.Sprintf("path not found: %s", path))
}

func (rt *Router) handleDeployments(method, ns, name string, body []byte) (int, []byte) {
	if name == "" {
		switch method {
		case http.MethodGet:
			items := rt.store.ListDeployments(ns)
			return jsonResponse(200, DeploymentList{
				TypeMeta: TypeMeta{Kind: "DeploymentList", APIVersion: "apps/v1"},
				Items:    items,
			})
		case http.MethodPost:
			var dep Deployment
			if err := json.Unmarshal(body, &dep); err != nil {
				return statusError(400, "BadRequest", err.Error())
			}
			dep.TypeMeta = TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"}
			dep.Namespace = ns
			if err := rt.store.CreateDeployment(&dep); err != nil {
				return statusError(409, "AlreadyExists", err.Error())
			}
			return jsonResponse(201, dep)
		}
	} else {
		switch method {
		case http.MethodGet:
			dep, ok := rt.store.GetDeployment(ns, name)
			if !ok {
				return statusError(404, "NotFound", fmt.Sprintf("deployments.apps %q not found", name))
			}
			return jsonResponse(200, dep)
		case http.MethodDelete:
			if err := rt.store.DeleteDeployment(ns, name); err != nil {
				return statusError(404, "NotFound", err.Error())
			}
			return jsonResponse(200, Status{TypeMeta: TypeMeta{Kind: "Status", APIVersion: "v1"}, Status: "Success", Code: 200})
		}
	}
	return statusError(405, "MethodNotAllowed", "method not allowed")
}

// --- Helpers ---

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}

func jsonResponse(status int, v interface{}) (int, []byte) {
	data, err := json.Marshal(v)
	if err != nil {
		return 500, []byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","message":"internal marshal error","code":500}`)
	}
	return status, data
}

func statusError(code int, reason, message string) (int, []byte) {
	return jsonResponse(code, Status{
		TypeMeta: TypeMeta{Kind: "Status", APIVersion: "v1"},
		Status:   "Failure",
		Message:  message,
		Reason:   reason,
		Code:     code,
	})
}
