package internal

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Store holds all k8s resources in memory, organized by resource type, namespace, and name.
type Store struct {
	mu              sync.RWMutex
	namespaces      map[string]*Namespace
	pods            map[string]map[string]*Pod            // ns -> name -> pod
	services        map[string]map[string]*Service        // ns -> name -> svc
	configMaps      map[string]map[string]*ConfigMap      // ns -> name -> cm
	secrets         map[string]map[string]*Secret         // ns -> name -> secret
	deployments     map[string]map[string]*Deployment     // ns -> name -> deploy
	nodes           map[string]*Node                      // name -> node
	resourceVersion atomic.Int64
	watchers        []chan WatchEvent
	watchersMu      sync.Mutex
}

// NewStore creates an empty k8s resource store with default namespace.
func NewStore() *Store {
	s := &Store{
		namespaces:  make(map[string]*Namespace),
		pods:        make(map[string]map[string]*Pod),
		services:    make(map[string]map[string]*Service),
		configMaps:  make(map[string]map[string]*ConfigMap),
		secrets:     make(map[string]map[string]*Secret),
		deployments: make(map[string]map[string]*Deployment),
		nodes:       make(map[string]*Node),
	}
	s.resourceVersion.Store(1)

	// Create default namespaces.
	for _, ns := range []string{"default", "kube-system", "kube-public", "kube-node-lease"} {
		s.namespaces[ns] = &Namespace{
			TypeMeta:   TypeMeta{Kind: "Namespace", APIVersion: "v1"},
			ObjectMeta: ObjectMeta{Name: ns, UID: generateUID(), ResourceVersion: "1", CreationTimestamp: Now()},
			Status:     NamespaceStatus{Phase: "Active"},
		}
	}

	// Create a default node.
	s.nodes["cloudmock-node"] = &Node{
		TypeMeta:   TypeMeta{Kind: "Node", APIVersion: "v1"},
		ObjectMeta: ObjectMeta{Name: "cloudmock-node", UID: generateUID(), ResourceVersion: "1", CreationTimestamp: Now()},
		Spec:       NodeSpec{PodCIDR: "10.244.0.0/24", ProviderID: "cloudmock://cloudmock-node"},
		Status: NodeStatus{
			Conditions: []NodeCondition{{Type: "Ready", Status: "True"}},
			Addresses:  []NodeAddress{{Type: "InternalIP", Address: "10.0.0.1"}, {Type: "Hostname", Address: "cloudmock-node"}},
		},
	}

	return s
}

func (s *Store) nextVersion() string {
	return strconv.FormatInt(s.resourceVersion.Add(1), 10)
}

func generateUID() string {
	// Simple sequential UIDs for mock purposes.
	return fmt.Sprintf("cm-%d", uidCounter.Add(1))
}

var uidCounter atomic.Int64

// --- Watch ---

// Subscribe returns a channel that receives watch events.
func (s *Store) Subscribe() chan WatchEvent {
	ch := make(chan WatchEvent, 100)
	s.watchersMu.Lock()
	s.watchers = append(s.watchers, ch)
	s.watchersMu.Unlock()
	return ch
}

// Unsubscribe removes a watcher channel.
func (s *Store) Unsubscribe(ch chan WatchEvent) {
	s.watchersMu.Lock()
	defer s.watchersMu.Unlock()
	for i, w := range s.watchers {
		if w == ch {
			s.watchers = append(s.watchers[:i], s.watchers[i+1:]...)
			close(ch)
			return
		}
	}
}

func (s *Store) notify(eventType string, obj interface{}) {
	s.watchersMu.Lock()
	watchers := make([]chan WatchEvent, len(s.watchers))
	copy(watchers, s.watchers)
	s.watchersMu.Unlock()

	event := WatchEvent{Type: eventType, Object: obj}
	for _, ch := range watchers {
		select {
		case ch <- event:
		default:
			// Drop event if watcher is slow.
		}
	}
}

// --- Namespaces ---

func (s *Store) GetNamespace(name string) (*Namespace, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ns, ok := s.namespaces[name]
	return ns, ok
}

func (s *Store) ListNamespaces() []Namespace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Namespace, 0, len(s.namespaces))
	for _, ns := range s.namespaces {
		out = append(out, *ns)
	}
	return out
}

func (s *Store) CreateNamespace(ns *Namespace) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.namespaces[ns.Name]; exists {
		return fmt.Errorf("namespace %q already exists", ns.Name)
	}
	ns.UID = generateUID()
	ns.ResourceVersion = s.nextVersion()
	ns.CreationTimestamp = Now()
	ns.Status.Phase = "Active"
	s.namespaces[ns.Name] = ns
	s.notify("ADDED", ns)
	return nil
}

func (s *Store) DeleteNamespace(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns, exists := s.namespaces[name]
	if !exists {
		return fmt.Errorf("namespace %q not found", name)
	}
	delete(s.namespaces, name)
	s.notify("DELETED", ns)
	return nil
}

// --- Pods ---

func (s *Store) GetPod(namespace, name string) (*Pod, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap, ok := s.pods[namespace]
	if !ok {
		return nil, false
	}
	pod, ok := nsMap[name]
	return pod, ok
}

func (s *Store) ListPods(namespace string) []Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap := s.pods[namespace]
	out := make([]Pod, 0, len(nsMap))
	for _, p := range nsMap {
		out = append(out, *p)
	}
	return out
}

func (s *Store) ListAllPods() []Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []Pod
	for _, nsMap := range s.pods {
		for _, p := range nsMap {
			out = append(out, *p)
		}
	}
	return out
}

func (s *Store) CreatePod(pod *Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := pod.Namespace
	if ns == "" {
		ns = "default"
		pod.Namespace = ns
	}
	if _, ok := s.namespaces[ns]; !ok {
		return fmt.Errorf("namespace %q not found", ns)
	}
	if s.pods[ns] == nil {
		s.pods[ns] = make(map[string]*Pod)
	}
	if _, exists := s.pods[ns][pod.Name]; exists {
		return fmt.Errorf("pod %q already exists in namespace %q", pod.Name, ns)
	}
	pod.UID = generateUID()
	pod.ResourceVersion = s.nextVersion()
	pod.CreationTimestamp = Now()
	now := Now()
	pod.Status.Phase = "Running"
	pod.Status.PodIP = fmt.Sprintf("10.244.0.%d", len(s.pods[ns])+2)
	pod.Status.HostIP = "10.0.0.1"
	pod.Status.StartTime = &now
	pod.Status.Conditions = []PodCondition{
		{Type: "Ready", Status: "True"},
		{Type: "PodScheduled", Status: "True"},
	}
	pod.Spec.NodeName = "cloudmock-node"
	s.pods[ns][pod.Name] = pod
	s.notify("ADDED", pod)
	return nil
}

func (s *Store) DeletePod(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	nsMap, ok := s.pods[namespace]
	if !ok {
		return fmt.Errorf("pod %q not found in namespace %q", name, namespace)
	}
	pod, exists := nsMap[name]
	if !exists {
		return fmt.Errorf("pod %q not found in namespace %q", name, namespace)
	}
	delete(nsMap, name)
	s.notify("DELETED", pod)
	return nil
}

// --- Services ---

func (s *Store) GetService(namespace, name string) (*Service, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap, ok := s.services[namespace]
	if !ok {
		return nil, false
	}
	svc, ok := nsMap[name]
	return svc, ok
}

func (s *Store) ListServices(namespace string) []Service {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap := s.services[namespace]
	out := make([]Service, 0, len(nsMap))
	for _, svc := range nsMap {
		out = append(out, *svc)
	}
	return out
}

func (s *Store) CreateService(svc *Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := svc.Namespace
	if ns == "" {
		ns = "default"
		svc.Namespace = ns
	}
	if s.services[ns] == nil {
		s.services[ns] = make(map[string]*Service)
	}
	if _, exists := s.services[ns][svc.Name]; exists {
		return fmt.Errorf("service %q already exists in namespace %q", svc.Name, ns)
	}
	svc.UID = generateUID()
	svc.ResourceVersion = s.nextVersion()
	svc.CreationTimestamp = Now()
	if svc.Spec.ClusterIP == "" {
		svc.Spec.ClusterIP = fmt.Sprintf("10.96.%d.%d", len(s.services)%256, len(s.services[ns])%256+1)
	}
	if svc.Spec.Type == "" {
		svc.Spec.Type = "ClusterIP"
	}
	s.services[ns][svc.Name] = svc
	s.notify("ADDED", svc)
	return nil
}

func (s *Store) DeleteService(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	nsMap, ok := s.services[namespace]
	if !ok {
		return fmt.Errorf("service %q not found in namespace %q", name, namespace)
	}
	svc, exists := nsMap[name]
	if !exists {
		return fmt.Errorf("service %q not found in namespace %q", name, namespace)
	}
	delete(nsMap, name)
	s.notify("DELETED", svc)
	return nil
}

// --- ConfigMaps ---

func (s *Store) GetConfigMap(namespace, name string) (*ConfigMap, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap, ok := s.configMaps[namespace]
	if !ok {
		return nil, false
	}
	cm, ok := nsMap[name]
	return cm, ok
}

func (s *Store) ListConfigMaps(namespace string) []ConfigMap {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap := s.configMaps[namespace]
	out := make([]ConfigMap, 0, len(nsMap))
	for _, cm := range nsMap {
		out = append(out, *cm)
	}
	return out
}

func (s *Store) CreateConfigMap(cm *ConfigMap) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := cm.Namespace
	if ns == "" {
		ns = "default"
		cm.Namespace = ns
	}
	if s.configMaps[ns] == nil {
		s.configMaps[ns] = make(map[string]*ConfigMap)
	}
	if _, exists := s.configMaps[ns][cm.Name]; exists {
		return fmt.Errorf("configmap %q already exists in namespace %q", cm.Name, ns)
	}
	cm.UID = generateUID()
	cm.ResourceVersion = s.nextVersion()
	cm.CreationTimestamp = Now()
	s.configMaps[ns][cm.Name] = cm
	s.notify("ADDED", cm)
	return nil
}

func (s *Store) DeleteConfigMap(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	nsMap, ok := s.configMaps[namespace]
	if !ok {
		return fmt.Errorf("configmap %q not found in namespace %q", name, namespace)
	}
	cm, exists := nsMap[name]
	if !exists {
		return fmt.Errorf("configmap %q not found in namespace %q", name, namespace)
	}
	delete(nsMap, name)
	s.notify("DELETED", cm)
	return nil
}

// --- Secrets ---

func (s *Store) GetSecret(namespace, name string) (*Secret, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap, ok := s.secrets[namespace]
	if !ok {
		return nil, false
	}
	sec, ok := nsMap[name]
	return sec, ok
}

func (s *Store) ListSecrets(namespace string) []Secret {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap := s.secrets[namespace]
	out := make([]Secret, 0, len(nsMap))
	for _, sec := range nsMap {
		out = append(out, *sec)
	}
	return out
}

func (s *Store) CreateSecret(sec *Secret) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := sec.Namespace
	if ns == "" {
		ns = "default"
		sec.Namespace = ns
	}
	if s.secrets[ns] == nil {
		s.secrets[ns] = make(map[string]*Secret)
	}
	if _, exists := s.secrets[ns][sec.Name]; exists {
		return fmt.Errorf("secret %q already exists in namespace %q", sec.Name, ns)
	}
	sec.UID = generateUID()
	sec.ResourceVersion = s.nextVersion()
	sec.CreationTimestamp = Now()
	if sec.Type == "" {
		sec.Type = "Opaque"
	}
	s.secrets[ns][sec.Name] = sec
	s.notify("ADDED", sec)
	return nil
}

func (s *Store) DeleteSecret(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	nsMap, ok := s.secrets[namespace]
	if !ok {
		return fmt.Errorf("secret %q not found in namespace %q", name, namespace)
	}
	sec, exists := nsMap[name]
	if !exists {
		return fmt.Errorf("secret %q not found in namespace %q", name, namespace)
	}
	delete(nsMap, name)
	s.notify("DELETED", sec)
	return nil
}

// --- Deployments ---

func (s *Store) GetDeployment(namespace, name string) (*Deployment, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap, ok := s.deployments[namespace]
	if !ok {
		return nil, false
	}
	dep, ok := nsMap[name]
	return dep, ok
}

func (s *Store) ListDeployments(namespace string) []Deployment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nsMap := s.deployments[namespace]
	out := make([]Deployment, 0, len(nsMap))
	for _, dep := range nsMap {
		out = append(out, *dep)
	}
	return out
}

func (s *Store) CreateDeployment(dep *Deployment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ns := dep.Namespace
	if ns == "" {
		ns = "default"
		dep.Namespace = ns
	}
	if s.deployments[ns] == nil {
		s.deployments[ns] = make(map[string]*Deployment)
	}
	if _, exists := s.deployments[ns][dep.Name]; exists {
		return fmt.Errorf("deployment %q already exists in namespace %q", dep.Name, ns)
	}
	dep.UID = generateUID()
	dep.ResourceVersion = s.nextVersion()
	dep.CreationTimestamp = Now()
	replicas := dep.Spec.Replicas
	if replicas == 0 {
		replicas = 1
	}
	dep.Status = DeploymentStatus{
		Replicas:          replicas,
		ReadyReplicas:     replicas,
		AvailableReplicas: replicas,
		UpdatedReplicas:   replicas,
	}
	s.deployments[ns][dep.Name] = dep
	s.notify("ADDED", dep)
	return nil
}

func (s *Store) DeleteDeployment(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	nsMap, ok := s.deployments[namespace]
	if !ok {
		return fmt.Errorf("deployment %q not found in namespace %q", name, namespace)
	}
	dep, exists := nsMap[name]
	if !exists {
		return fmt.Errorf("deployment %q not found in namespace %q", name, namespace)
	}
	delete(nsMap, name)
	s.notify("DELETED", dep)
	return nil
}

// --- Nodes ---

func (s *Store) GetNode(name string) (*Node, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[name]
	return node, ok
}

func (s *Store) ListNodes() []Node {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		out = append(out, *node)
	}
	return out
}

// MatchLabels returns true if the resource labels contain all selector labels.
func MatchLabels(resourceLabels, selector map[string]string) bool {
	for k, v := range selector {
		if resourceLabels[k] != v {
			return false
		}
	}
	return true
}

// FilterPodsByLabels parses a k8s label selector string and filters pods.
func (s *Store) FilterPodsByLabels(namespace string, selector string) []Pod {
	pods := s.ListPods(namespace)
	if selector == "" {
		return pods
	}
	parsed := parseLabelSelector(selector)
	var out []Pod
	for _, pod := range pods {
		if MatchLabels(pod.Labels, parsed) {
			out = append(out, pod)
		}
	}
	return out
}

func parseLabelSelector(sel string) map[string]string {
	result := make(map[string]string)
	for _, part := range strings.Split(sel, ",") {
		part = strings.TrimSpace(part)
		if kv := strings.SplitN(part, "=", 2); len(kv) == 2 {
			result[kv[0]] = kv[1]
		}
	}
	return result
}
