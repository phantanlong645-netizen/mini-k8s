package store

import "mini-k8s/pkg/api"

type Store interface {
	//about Pod
	CreatePod(pod *api.Pod) error
	GetPod(namespace, name string) (*api.Pod, error)
	ListPods(namesapce string) ([]*api.Pod, error)
	DeletePod(namespace, name string) error

	//about Node
	CreateNode(node *api.Node) error
	GetNode(name string) (*api.Node, error)
	ListNodes() ([]*api.Node, error)
}
