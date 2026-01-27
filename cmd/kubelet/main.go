package main

import (
	"flag"
	"log"
	"mini-k8s/pkg/api"
	"time"
)

const DefaultNamespace = "default"

type Kubelet struct {
	NodeName    string `json:"nodeName"`
	NodeAddress string `json:"nodeAddress"`
	APIclient   *api.Client
}

func NewKubelet(name string, address string, apiserverURl string) (*Kubelet, error) {

	client, err := api.NewClient(apiserverURl)
	if err != nil {
		log.Printf("Error creating  client: %s", err)
		return nil, err
	}
	return &Kubelet{
		NodeName:    name,
		NodeAddress: address,
		APIclient:   client,
	}, nil
}

func (kubelet *Kubelet) registerNode() error {
	node := &api.Node{
		Name:    kubelet.NodeName,
		Address: kubelet.NodeAddress,
		Status:  api.NodeReady,
	}
	createNode, err := kubelet.APIclient.CreateNode(node)
	//kubelet是无状态的，如果重启了 注册节点并不意味着 系统出问题了 可能是已经注册过了
	//场景：节点重启
	//物理服务器重启
	//Kubelet 自动启动
	//Kubelet 尝试创建节点（但节点已存在）
	//创建失败 → 更新节点状态为 Ready
	//系统恢复正常
	if err != nil {
		log.Printf("Failed to register node %s, attempting to update: %v", kubelet.NodeName, err)
		if errUpdate := kubelet.APIclient.UpdateNode(node); errUpdate != nil {
			log.Printf("Error updating node: %s", errUpdate)
			return errUpdate
		}
		log.Printf("Node %s updated successfully after initial registration failure.", kubelet.NodeName)
		return nil
	}
	log.Printf("Node %s registered successfully with address %s and status %s", createNode.Name, createNode.Address, createNode.Status)
	return nil
}
func (kubelet *Kubelet) syncPods() {
	log.Printf("[%s] syncing pods", kubelet.NodeName)
	pods, err := kubelet.APIclient.ListPods(DefaultNamespace, "")
	if err != nil {
		log.Printf("Error listing pods: %s", err)
		return
	}
	for _, pod := range pods {
		//先检查这个pod 是不是属于这个NOde
		if pod.NodeName == kubelet.NodeName {
			//检查这个pod是不是属于被删除状态
			if pod.DeletionTimestamp != nil {
				//一旦有了 DeletionTimestamp，Pod 通常会处于以下两个阶段之一：
				//PodTerminating 或 PodDeleting 阶段： 这是删除标记刚被打上时的状态。此时 Kubelet 会观察到这个标记，开
				//始在本地执行“关机”操作（停止容器、释放网络等）。
				//
				//任何运行中的状态（Running/Pending）： 即便 Pod 正在 Running，
				//只要 DeletionTimestamp 一出现，它的逻辑身份就立刻变成了“待销毁”。Kubelet
				//必须停止一切正常业务，转而处理终止逻辑。
				if pod.Phase != api.PodSucceeded && pod.Phase != api.PodDeleted && pod.Phase != api.PodFailed {
					log.Printf("[%s] Detected terminating pod %s. Simulating cleanup and marking as Deleted.", kubelet.NodeName, pod.Name)
					updatePod := pod
					//执行物理上的消除
					updatePod.Phase = api.PodDeleted
					if err := kubelet.APIclient.UpdatePod(&updatePod); err != nil {
						log.Printf("[%s] Error updating pod %s to Deleted after termination: %v", kubelet.NodeName, pod.Name, err)
					} else {
						log.Printf("[%s] Pod %s marked as Deleted after termination processing.", kubelet.NodeName, pod.Name)
					}
				} else {
					log.Printf("[%s] Pod %s is terminating and already in state %s. No Kubelet action needed.", kubelet.NodeName, pod.Name, pod.Phase)
				}
				continue
			}

			switch pod.Phase {
			case api.PodScheduled:
				log.Printf("[%s] Found scheduled pod %s. 'Starting' it...", kubelet.NodeName, pod.Name)
				updatePod := pod
				//使用容器运行时 拉镜像 跑起来....
				updatePod.Phase = api.PodRunning
				if err := kubelet.APIclient.UpdatePod(&updatePod); err != nil {
					log.Printf("[%s] Error updating pod %s to Running: %v", kubelet.NodeName, pod.Name, err)
				} else {
					log.Printf("[%s] Pod %s with image '%s' is now 'Running'.", kubelet.NodeName, pod.Name, pod.Image)
				}

			case api.PodRunning:
				break

			case api.PodTerminating:
				log.Printf("[%s] Pod %s found in Terminating phase. Processing termination.", kubelet.NodeName, pod.Name)
				if pod.Phase != api.PodSucceeded && pod.Phase != api.PodFailed && pod.Phase != api.PodDeleted {
					updatePod := pod
					updatePod.Phase = api.PodDeleted
					if err := kubelet.APIclient.UpdatePod(&updatePod); err != nil {
						log.Printf("[%s] Error updating pod %s from Terminating to Deleted: %v", kubelet.NodeName, pod.Name, err)
					} else {
						log.Printf("[%s] Pod %s (in Terminating phase) marked as Deleted.", kubelet.NodeName, pod.Name)
					}
				}
			case api.PodDeleting:
				log.Printf("[%s] Detected pod %s in PodDeleting phase. Handling as terminating.", kubelet.NodeName, pod.Name)
				if pod.DeletionTimestamp == nil {
					log.Printf("[%s] Warning: Pod %s in PodDeleting phase but DeletionTimestamp is nil. This should be synchronized.", kubelet.NodeName, pod.Name)
				}
				if pod.Phase != api.PodSucceeded && pod.Phase != api.PodFailed {
					updatedPod := pod
					updatedPod.Phase = api.PodSucceeded
					if err := kubelet.APIclient.UpdatePod(&updatedPod); err != nil {
						log.Printf("[%s] Error updating pod %s from PodDeleting to Succeeded: %v", kubelet.NodeName, pod.Name, err)
					} else {
						log.Printf("[%s] Pod %s (in PodDeleting phase) marked as Succeeded.", kubelet.NodeName, pod.Name)
					}
				}

			default:
				if pod.Phase != api.PodRunning && pod.Phase != api.PodSucceeded && pod.Phase != api.PodFailed {
					log.Printf("[%s] Pod %s found in unhandled phase: %s", kubelet.NodeName, pod.Name, pod.Phase)

				}

			}

		}
	}

}
func main() {
	nodeName := flag.String("name", "", "Name of this node (kubelet)")
	nodeAddress := flag.String("address", "localhost:10250", "Address of this node (e.g. IP or hostname, port is informational for mock)")
	apiServerURL := flag.String("apiserver", "http://localhost:8055", "URL of the API server")
	syncInterval := flag.Duration("sync-interval", 10*time.Second, "Pod synchronization interval")
	flag.Parse()
	if *nodeName == "" {
		log.Fatalf("Node name must be specified using -name flag")
	}
	log.Printf("Kubelet for node '%s' starting. Node address: %s. API Server: %s", *nodeName, *nodeAddress, *apiServerURL)
	kubelet, err := NewKubelet(*nodeName, *nodeAddress, *apiServerURL)
	if err != nil {
		log.Fatalf("Failed to create Kubelet: %v", err)
	}
	if err := kubelet.registerNode(); err != nil {
		log.Fatalf("Failed to register node with API server: %v. Ensure API server is running.", err)
	}

	log.Printf("Kubelet for node '%s' registered. Starting pod sync loop with interval %v.", *nodeName, *syncInterval)
	for {
		kubelet.syncPods()
		time.Sleep(*syncInterval)
	}

}
