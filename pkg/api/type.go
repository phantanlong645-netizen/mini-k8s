package api

import "time"

const (
	PodPending     PodPhase = "Pending"   // The pod has been accepted by the system, but one or more of the container images has not been created. This includes time before being scheduled as well as time spent downloading images over the network.
	PodScheduled   PodPhase = "Scheduled" // The pod has been scheduled to a node, but is not yet running.
	PodRunning     PodPhase = "Running"   // The pod has been bound to a node, and all of the containers have been created. At least one container is still running, or is in the process of starting or restarting.
	PodDeleted     PodPhase = "Deleted"   // The pod's resources have been reclaimed by the Kubelet. This is a final state.
	PodSucceeded   PodPhase = "Succeeded" // All containers in the pod have terminated in success, and will not be restarted.
	PodFailed      PodPhase = "Failed"    // All containers in the pod have terminated, and at least one container has terminated in failure. The container either exited with non-zero status or was terminated by the system.
	PodDeleting    PodPhase = "Deleting"  // The pod is marked for deletion.
	PodTerminating PodPhase = "Terminating"
)

type Pod struct {
	Name              string     `json:"name"`
	Namespace         string     `json:"namespace"`
	Image             string     `json:"image"`
	NodeName          string     `json:"nodeName"`
	Phase             PodPhase   `json:"phase"`                       //跟踪容器在其生命周期中的状态：待处理、已调度、正在运行、终止中、已删除等
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"` //启用软删除功能，以便 pod 能被优雅地清理
}
type PodPhase string

type Node struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	Status  string `json:"status"`
}
