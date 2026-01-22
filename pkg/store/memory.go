package store

import (
	"fmt"
	"mini-k8s/pkg/api"
	"sync"
	"time"
)

type InMemoryStore struct {
	mu    sync.RWMutex
	pods  map[string]*api.Pod
	nodes map[string]*api.Node
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		pods:  make(map[string]*api.Pod),
		nodes: make(map[string]*api.Node),
	}
}
func (ms *InMemoryStore) CreatePod(pod *api.Pod) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	//检查是否存在，存在则返回错误，不存在则新加
	key := fmt.Sprintf("%s%s", pod.Namespace, pod.Name)
	if _, ok := ms.pods[key]; ok {
		return fmt.Errorf("pod %s already exists", key)
	} else {
		ms.pods[key] = pod
	}
	return nil
}
func (ms *InMemoryStore) GetPod(namespace, name string) (*api.Pod, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	key := fmt.Sprintf("%s%s", namespace, name)
	pod, ok := ms.pods[key]
	if !ok {
		return nil, fmt.Errorf("pod %s not found", key)
	}
	return pod, nil
}
func (ms *InMemoryStore) UpdatePod(pod *api.Pod) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	key := fmt.Sprintf("%s%s", pod.Namespace, pod.Name)
	existingpod, ok := ms.pods[key]
	if !ok {
		return fmt.Errorf("pod %s not found", key)
	}
	//如果这个pod是正在被删除的过程中，那pod是新传入的状态，假如新传入的状态又显示没删除或者删除的时间戳不一致，那就是错误的更新，应该返回错误
	if existingpod.DeletionTimestamp != nil {
		if pod.DeletionTimestamp == nil || !pod.DeletionTimestamp.Equal(*existingpod.DeletionTimestamp) {
			return fmt.Errorf("cannot update pod %s in namespace %s: incoming update does not have matching DeletionTimestamp for an already terminating pod", pod.Name, pod.Namespace)
		}
		//如果pod是已经在终止的过程了，那可以更改为像failed的状态
		if pod.Phase == api.PodSucceeded || pod.Phase == api.PodFailed || pod.Phase == api.PodTerminating || pod.Phase == api.PodDeleted {
			if pod.Name != existingpod.Name {
				return fmt.Errorf("cannot update pod %s in namespace %s: the pod is terminating", pod.Name, pod.Namespace)
			}
			ms.pods[key] = pod
			return nil
		}
		return fmt.Errorf("cannot update pod %s in namespace %s to phase %s as it is terminating; only Succeeded, Failed, or Terminating are allowed", pod.Name, pod.Namespace, pod.Phase)

	}
	//这个条件代表你想吧这个pod设置为删除状态，但是这是更新函数  根据职责单一原则，你要去delete函数去执行
	if existingpod.DeletionTimestamp == nil && pod.DeletionTimestamp != nil {
		return fmt.Errorf("to mark pod %s in namespace %s for deletion, use DeletePod method", pod.Name, pod.Namespace)
	}
	ms.pods[key] = pod
	return nil
}
func (ms *InMemoryStore) DeletePod(namespace, name string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	//1 假如说这个pod还在运行状态那就可以改为删除状态
	//2 如果已经是删除状态了 那可以不设置了
	key := fmt.Sprintf("%s%s", namespace, name)
	pod, ok := ms.pods[key]
	if !ok {
		return fmt.Errorf("pod %s not found", key)
	}
	if pod.DeletionTimestamp != nil {
		return fmt.Errorf(" pod %s in namespace %s is terminating", pod.Name, pod.Namespace)
	}
	now := time.Now()
	pod.DeletionTimestamp = &now
	pod.Phase = api.PodTerminating
	ms.pods[key] = pod
	return nil
}

func (ms *InMemoryStore) ListPods(namespace string) ([]*api.Pod, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	var result []*api.Pod
	for _, pod := range ms.pods {
		if pod.Namespace == namespace {
			result = append(result, pod)
		}
	}
	return result, nil
}

func (ms *InMemoryStore) CreateNode(node *api.Node) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	_, ok := ms.nodes[node.Name]
	if !ok {
		ms.nodes[node.Name] = node
		return nil
	}

	return fmt.Errorf("node %s already exists", node.Name)
}
func (ms *InMemoryStore) GetNode(name string) (*api.Node, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	existingNode, ok := ms.nodes[name]
	if !ok {
		return nil, fmt.Errorf("node %s not found", name)
	}
	return existingNode, nil
}
func (ms *InMemoryStore) UpdateNode(node *api.Node) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	_, ok := ms.nodes[node.Name]
	if !ok {
		return fmt.Errorf("node %s not found", node.Name)
	}
	ms.nodes[node.Name] = node
	return nil
}
func (s *InMemoryStore) DeleteNode(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[name]; !exists {
		return fmt.Errorf("node %s not found for deletion", name)
	}
	delete(s.nodes, name)
	return nil
}
func (ms *InMemoryStore) ListNodes() ([]*api.Node, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	var result []*api.Node
	for _, node := range ms.nodes {
		result = append(result, node)
	}
	return result, nil
}
