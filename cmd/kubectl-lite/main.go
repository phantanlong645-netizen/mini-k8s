package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mini-k8s/pkg/api"
	"os"
	"strings"
)

const DefaultNamespace = "default"

func main() {
	apiServerURL := flag.String("apiserver", "http://localhost:8055", "URL of the API server")
	flag.Parse()
	if len(flag.Args()) < 1 {
		fmt.Println("Error: No command specified.")
		printUsage()
		os.Exit(1)
	}
	client, err := api.NewClient(*apiServerURL)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	command := flag.Args()[0]
	args := flag.Args()[1:]

	switch command {
	case "create":
		handleCreateCommand(client, args)
	case "get":
		handleGetCommand(client, args)
	case "delete":
		handleDeleteCommand(client, args)
	case "register":
		handleRegisterNodeCommand(client, args)
	default:
		fmt.Println("Error: Unknown command.")
		printUsage()
		os.Exit(1)
	}
}
func printUsage() {
	fmt.Println("Usage: kubectl-lite --apiserver <url> <command> <subcommand> [flags]")
	fmt.Println("Commands:")
	fmt.Println("  create pod --name <name> --image <image> [--namespace <ns>]")
	fmt.Println("  get pods [--namespace <ns>]")
	fmt.Println("  get pod <name> [--namespace <ns>]")
	fmt.Println("  get nodes")
	fmt.Println("  get node <name>")
	fmt.Println("  delete pod <name> [--namespace <ns>]")
	fmt.Println("  register node --name <name> --address <addr>")
	fmt.Println("Global flags:")
	fmt.Println("  --apiserver <url>  URL of the API server (default: http://localhost:8055)")
}
func handleCreateCommand(client *api.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: kubectl-lite create <resource_type> [flags]")
		fmt.Println("Example: kubectl-lite create pod --name mypod --image nginx")
		os.Exit(1)
	}
	resourceType := args[0]
	commandArgs := args[1:]
	switch resourceType {
	case "pod":
		createPodCmd := flag.NewFlagSet("create pod", flag.ExitOnError)
		podName := createPodCmd.String("name", "", "Name of the pod")
		podImage := createPodCmd.String("image", "", "Image to use for the pod")
		podNamespace := createPodCmd.String("namespace", "", "Namespace of the pod")
		if err := createPodCmd.Parse(commandArgs); err != nil {
			fmt.Printf("Error parsing 'create pod' flags: %v\n", err)
			os.Exit(1)
		}
		if *podName == "" || *podImage == "" {
			fmt.Println("Error: --name and --image are required for creating a pod")
			createPodCmd.Usage()
			os.Exit(1)
		}
		pod := api.Pod{
			Name:      *podName,
			Image:     *podImage,
			Namespace: *podNamespace,
		}
		createdPod, err := client.CreatePod(*podNamespace, &pod)
		if err != nil {
			fmt.Printf("Error creating pod: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Pod %s/%s created\n\n", createdPod.Namespace, createdPod.Name)
	default:
		fmt.Printf("Error: Unknown resource type for create: %s\n", resourceType)
		fmt.Println("Supported resource types for create: pod")
		os.Exit(1)
	}

}
func handleGetCommand(client *api.Client, args []string) {
	getPodCmd := flag.NewFlagSet("get pod", flag.ExitOnError)
	PodNamespace := getPodCmd.String("namespace", DefaultNamespace, "Namespace of the pod")
	if len(args) < 1 {
		fmt.Println("Usage: kubectl-lite create <resource_type> [flags]")
		fmt.Println("Example: kubectl-lite create pod --name mypod ")
		os.Exit(1)
	}
	resourceType := args[0]
	var resourceName string
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		resourceName = args[1]
		getPodCmd.Parse(args[2:])
	} else {
		getPodCmd.Parse(args[1:])
	}
	switch resourceType {
	case "pod", "pods":

		if resourceName == "" {
			pods, err := client.ListPods(*PodNamespace, "")
			if err != nil {
				fmt.Printf("Error listing pods: %v\n", err)
				os.Exit(1)
			}
			prettyPrint(pods)
		} else {
			pod, err := client.GetPod(*PodNamespace, resourceName)
			if err != nil {
				fmt.Printf("Error getting pod: %v\n", err)
				os.Exit(1)
			}
			prettyPrint(pod)
		}
	case "nodes", "node":
		if resourceName == "" {
			nodes, err := client.ListNodes("")
			if err != nil {
				fmt.Printf("Error listing nodes: %v\n", err)
				os.Exit(1)
			}
			prettyPrint(nodes)
		} else {
			node, err := client.GetNode(resourceName)
			if err != nil {
				fmt.Printf("Error getting node: %v\n", err)
				os.Exit(1)
			}
			prettyPrint(node)
		}
	default:
		fmt.Printf("Unknown resource type for get: %s\n", resourceType)
		os.Exit(1)
	}
}

func handleDeleteCommand(client *api.Client, args []string) {
	deleteCmd := flag.NewFlagSet("delete ", flag.ExitOnError)
	podnamespace := deleteCmd.String("namespace", DefaultNamespace, "Namespace of the pod")
	if len(args) < 2 {
		fmt.Println("Usage: kubectl-lite delete <resource_type> [flags]")
		os.Exit(1)
	}
	resourceType := args[0]
	resourceName := args[1]
	deleteCmd.Parse(args[2:])
	switch resourceType {
	case "pod":
		if resourceName == "" {
			fmt.Println("Error: --name or --namespace is required")
			os.Exit(1)
		} else {
			err := client.DeletePod(*podnamespace, resourceName)
			if err != nil {
				fmt.Printf("Error deleting pod: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Pod %s/%s deleted\n\n", *podnamespace, resourceName)
		}
	default:
		fmt.Printf("Unknown resource type for delete: %s\n", resourceType)
		os.Exit(1)
	}
}
func handleRegisterNodeCommand(client *api.Client, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: kubectl-lite register <resource_type> [flags]")
		os.Exit(1)
	}
	resourceType := args[0]
	commandArgs := args[1:]
	if resourceType != "node" {
		fmt.Printf("Unknown resource type for register: %s\n", resourceType)
		os.Exit(1)
	}
	registerNodeCmd := flag.NewFlagSet("register node", flag.ExitOnError)
	nodeName := registerNodeCmd.String("name", "", "Name of the node")
	nodeAddress := registerNodeCmd.String("address", "", "Address of the node (e.g. IP)")

	if err := registerNodeCmd.Parse(commandArgs); err != nil {
		fmt.Printf("Error parsing 'register node' flags: %v\n", err)
		os.Exit(1)
	}

	if *nodeName == "" || *nodeAddress == "" {
		fmt.Println("Error: --name and --address are required for registering a node")
		registerNodeCmd.Usage()
		os.Exit(1)
	}

	node := &api.Node{Name: *nodeName, Address: *nodeAddress, Status: "Ready"} // Assuming Address field exists in api.Node
	createdNode, err := client.CreateNode(node)
	if err != nil {
		log.Fatalf("Error registering node: %v", err)
	}
	fmt.Printf("Node %s registered with address %s\n", createdNode.Name, createdNode.Address)

}

func prettyPrint(data interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", " ")
	if err := enc.Encode(data); err != nil {
		log.Fatal(err)
	}
}
