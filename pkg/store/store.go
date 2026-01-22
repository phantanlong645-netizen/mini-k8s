package store

import "mini-k8s/pkg/api"

type Store interface {
	//about Pod
	CreatePod(pod *api.Pod) error
	GetPod(namespace, name string) (*api.Pod, error)
	UpdatePod(pod *api.Pod) error
	DeletePod(namespace, name string) error
	ListPods(namespace string) ([]*api.Pod, error)

	// Node operations
	CreateNode(node *api.Node) error
	GetNode(name string) (*api.Node, error)
	UpdateNode(node *api.Node) error
	DeleteNode(name string) error
	ListNodes() ([]*api.Node, error)
}
