package apiserver

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"mini-k8s/pkg/api"
	"mini-k8s/pkg/store"
	"strings"
)

const DefaultNamespace = "default"

type APIServer struct {
	store store.Store
}

func NewAPIServer(s store.Store) *APIServer {
	return &APIServer{s}
}
func (s *APIServer) Serve(port string) {
	router := gin.Default() // Use Gin router

	// Pod routes
	// /api/v1/namespaces/{namespace}/pods
	podsGroup := router.Group("/api/v1/namespaces/:namespace/pods")
	{
		podsGroup.POST("", s.createPodHandlerGin)
		podsGroup.GET("", s.listPodsHandlerGin)
		podsGroup.GET("/:podname", s.getPodHandlerGin)
		podsGroup.PUT("/:podname", s.updatePodHandlerGin) // Added route for updating a pod
		podsGroup.DELETE("/:podname", s.deletePodHandlerGin)
	}

	// Node routes
	// /api/v1/nodes
	nodesGroup := router.Group("/api/v1/nodes")
	{
		nodesGroup.POST("", s.createNodeHandlerGin)
		nodesGroup.GET("", s.listNodesHandlerGin)
		nodesGroup.GET("/:nodename", s.getNodeHandlerGin)
		nodesGroup.PUT("/:nodename", s.updateNodeHandlerGin) // Add PUT route for updating a node
		// DELETE for a node could be added here: nodesGroup.DELETE("/:nodename", s.deleteNodeHandlerGin)
	}

	log.Printf("API Server starting on port %s using Gin", port)
	// if err := http.ListenAndServe(":"+port, mux); err != nil { // Old http way
	if err := router.Run(":" + port); err != nil { // Gin way
		log.Fatalf("Failed to start Gin server: %v", err)
	}
}
func (s *APIServer) createPodHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	var pod api.Pod
	if err := c.ShouldBindJSON(&pod); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body" + err.Error()})
		return
	}
	if pod.Name == "" {
		c.JSON(400, gin.H{"error": "Pod name must be provided"})
	}
	pod.Namespace = namespace
	if pod.Namespace == "" {
		pod.Namespace = DefaultNamespace
	}
	pod.Phase = api.PodPending
	pod.NodeName = ""
	if err := s.store.CreatePod(&pod); err != nil {
		log.Printf("Error creating pod %s/%s in store: %v", pod.Namespace, pod.Name, err) // Log the actual error
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(409, gin.H{"error": "failed to create pod because:" + err.Error()})
		} else {
			c.JSON(500, gin.H{"error": "Failed to create pod: " + err.Error()}) // 500 for other errors
		}
		return
	}
	log.Printf("created pod %s/%s", pod.Namespace, pod.Name)
	c.JSON(201, pod)
}
func (s *APIServer) getPodHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	podName := c.Param("podname")
	pod, err := s.store.GetPod(namespace, podName)
	if err != nil {
		c.JSON(404, gin.H{"error": "Pod not found: " + err.Error()})
		return
	}
	c.JSON(200, pod)
}
func (s *APIServer) listPodsHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	pods, err := s.store.ListPods(namespace)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to list pods: " + err.Error()})
		return
	}
	c.JSON(200, pods)
}
func (s *APIServer) deletePodHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	podName := c.Param("podname")
	if err := s.store.DeletePod(namespace, podName); err != nil {
		log.Printf("Error deleting pod %s/%s from store: %v", namespace, podName, err) // Log the actual error
		if strings.Contains(err.Error(), "not found") {
			c.JSON(404, gin.H{"error": "Failed to delete pod: " + err.Error()}) // 404 Not Found
		} else {
			c.JSON(500, gin.H{"error": "Failed to delete pod: " + err.Error()}) // 500 for other errors
		}
		return
	}
	log.Printf("Deleted pod %s/%s", namespace, podName)
	c.JSON(200, gin.H{"message": fmt.Sprintf("Pod %s/%s deleted", namespace, podName)})
}

func (s *APIServer) updatePodHandlerGin(c *gin.Context) {
	namespace := c.Param("namespace")
	podName := c.Param("podname")

	var pod api.Pod
	if err := c.ShouldBindJSON(&pod); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if pod.Name != podName {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Pod name in body (%s) does not match name in URL (%s)", pod.Name, podName)})
		return
	}
	if pod.Namespace != namespace {
		c.JSON(400, gin.H{"error": fmt.Sprintf("Pod namespace in body (%s) does not match namespace in URL (%s)", pod.Namespace, namespace)})
		return
	}

	// Ensure the pod exists before updating (optional, store might handle this)
	_, err := s.store.GetPod(namespace, podName)
	if err != nil {
		c.JSON(404, gin.H{"error": fmt.Sprintf("Pod %s/%s not found for update: %s", namespace, podName, err.Error())})
		return
	}

	if err := s.store.UpdatePod(&pod); err != nil {
		log.Printf("Failed to update pod in store: %v", err)
		c.JSON(500, gin.H{"error": "Failed to update pod: " + err.Error()})
		return
	}

	c.JSON(200, pod)
}
func (s *APIServer) createNodeHandlerGin(c *gin.Context) {
	var node api.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
	}
	if node.Name == "" {
		c.JSON(400, gin.H{"error": "Node name must be provided"})
	}
	if node.Status == "" {
		node.Status = api.NodeNotReady
	}
	if err := s.store.CreateNode(&node); err != nil {
		c.JSON(500, gin.H{"error": "Failed to create node: " + err.Error()})
	}
	c.JSON(201, node)
}

// Gin handler for getting a specific node
func (s *APIServer) getNodeHandlerGin(c *gin.Context) {
	nodeName := c.Param("nodename")
	node, err := s.store.GetNode(nodeName)
	if err != nil {
		c.JSON(404, gin.H{"error": "Node not found: " + err.Error()})
		return
	}
	c.JSON(200, node)
}

// Gin handler for listing all nodes
func (s *APIServer) listNodesHandlerGin(c *gin.Context) {
	nodes, err := s.store.ListNodes()
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to list nodes: " + err.Error()})
		return
	}
	c.JSON(200, nodes)
}
func (s *APIServer) updateNodeHandlerGin(c *gin.Context) {
	nodeName := c.Param("nodename")
	var updateNode api.Node
	if err := c.ShouldBindJSON(&updateNode); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}
	if updateNode.Name != "" && updateNode.Name != nodeName {
		c.JSON(400, gin.H{"error": "Node name in body does not match node name"})
		return
	}
	updateNode.Name = nodeName

	_, err := s.store.GetNode(nodeName)
	if err != nil {
		c.JSON(404, gin.H{"error": "Node not found: " + err.Error()})
		return
	}
	if err := s.store.UpdateNode(&updateNode); err != nil {
		c.JSON(500, gin.H{"error": "Failed to update node: " + err.Error()})
		return
	}
	log.Printf("update node %s", nodeName)
	c.JSON(200, updateNode)

}
func main() {
	port := flag.String("port", "8080", "Port to run the api server on")
	flag.Parse()
	gin.SetMode(gin.ReleaseMode)
	dataStore := store.NewInMemoryStore()
	server := NewAPIServer(dataStore)
	server.Serve(*port)

}
