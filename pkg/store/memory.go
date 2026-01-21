package store

import (
	"fmt"
	"mini-k8s/pkg/api"
	"sync"
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
