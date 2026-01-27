package main

import (
	"flag"
	"log"
	"mini-k8s/pkg/api"
	"time"
)

const DefaultNamespace = "default" //如果不标注就去遍历默认的
var nextNodeIndex = 0

// 调度器的主函数，负责获取待调度的Pod并分配到就绪节点
func schedulePods(Client *api.Client) {
	pendingPods, err := Client.ListPods(DefaultNamespace, api.PodPending)
	if err != nil {
		log.Printf("Failed to fetch pending pods: %s", err)
		return
	}
	if len(pendingPods) == 0 {
		log.Printf("No pending pods found")
		return
	}
	log.Printf("Found %d pending pods", len(pendingPods))
	readyNodes, err := Client.ListNodes(api.NodeReady)
	if err != nil {
		log.Printf("Error fetching ready nodes: %s", err)
		return
	}
	if len(readyNodes) == 0 {
		log.Printf("No ready nodes found")
		return
	}
	log.Printf("Found %d ready nodes", len(readyNodes))
	for _, pod := range pendingPods {
		if pod.DeletionTimestamp != nil {
			log.Printf("Pod %s is being deleted", pod.Name)
			continue
		}
		if len(readyNodes) == 0 {
			log.Printf("No ready nodes available to schedule pod %s", pod.Name)
			continue
		}
		selectedNode := readyNodes[nextNodeIndex%len(readyNodes)]
		nextNodeIndex++

		podToudpdate := pod
		podToudpdate.NodeName = selectedNode.Name
		podToudpdate.Phase = api.PodScheduled

		log.Printf(" attempt  Scheduling pod %s to node %s", pod.Name, selectedNode.Name)

		if err := Client.UpdatePod(&podToudpdate); err != nil {
			log.Printf("Error updating pod %s to UDP node: %s", pod.Name, err)
		} else {
			log.Printf(" successfully Updated pod %s to UDP node: %s", pod.Name, selectedNode.Name)
		}
	}
}

func main() {
	apiServerURL := flag.String("apiserver", "http://localhost:8055", "URL of the API server")
	scheduleInterval := flag.Duration("interval", 5*time.Second, "Interval between scheduling pods")
	flag.Parse()
	log.Printf("Starting Scheduler with URL %s", *apiServerURL)
	client, err := api.NewClient(*apiServerURL)
	if err != nil {
		log.Fatalf("Error creating client: %s", err)
	}
	log.Printf("Scheduler created with URL %s", *apiServerURL)
	for {
		schedulePods(client)
		time.Sleep(*scheduleInterval)
	}

}
