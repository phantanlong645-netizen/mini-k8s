package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(baseURLStr string) (*Client, error) {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("parsing base URL: %w", err)
	}
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (c *Client) buildURL(pathSegments ...string) string {
	finalPath := c.baseURL.Path
	for _, segment := range pathSegments {
		finalPath = fmt.Sprintf("%s/%s", finalPath, segment)
	}
	u := *c.baseURL
	u.Path = finalPath
	return u.String()
}

func decodeAPIError(resp *http.Response) error {
	var payload struct {
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err == nil && payload.Error != "" {
		return fmt.Errorf("%s", payload.Error)
	}
	return fmt.Errorf("server returned %d %s", resp.StatusCode, resp.Status)
}

func (c *Client) CreatePod(namespace string, pod *Pod) (*Pod, error) {
	if namespace == "" {
		namespace = "default"
	}
	urlStr := c.buildURL("api", "v1", "namespaces", namespace, "pods")

	body, err := json.Marshal(pod)
	if err != nil {
		return nil, fmt.Errorf("marshalling pod: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, urlStr, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, decodeAPIError(resp)
	}

	var created Pod
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &created, nil
}

func (c *Client) GetPod(namespace, name string) (*Pod, error) {
	if namespace == "" {
		namespace = "default"
	}
	urlStr := c.buildURL("api", "v1", "namespaces", namespace, "pods", name)
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIError(resp)
	}

	var pod Pod
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &pod, nil
}

func (c *Client) ListPods(namespace string, phase PodPhase) ([]Pod, error) {
	if namespace == "" {
		namespace = "default"
	}
	urlStr := c.buildURL("api", "v1", "namespaces", namespace, "pods")
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIError(resp)
	}

	var all []Pod
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if phase == "" {
		return all, nil
	}
	out := make([]Pod, 0, len(all))
	for _, p := range all {
		if p.Phase == phase {
			out = append(out, p)
		}
	}
	return out, nil
}

func (c *Client) UpdatePod(pod *Pod) error {
	if pod == nil || pod.Name == "" {
		return fmt.Errorf("pod name must be specified for update")
	}
	if pod.Namespace == "" {
		pod.Namespace = "default"
	}
	urlStr := c.buildURL("api", "v1", "namespaces", pod.Namespace, "pods", pod.Name)

	body, err := json.Marshal(pod)
	if err != nil {
		return fmt.Errorf("marshalling pod: %w", err)
	}
	req, err := http.NewRequest(http.MethodPut, urlStr, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}
	return nil
}

func (c *Client) DeletePod(namespace, name string) error {
	if namespace == "" {
		namespace = "default"
	}
	urlStr := c.buildURL("api", "v1", "namespaces", namespace, "pods", name)
	req, err := http.NewRequest(http.MethodDelete, urlStr, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return decodeAPIError(resp)
	}
	return nil
}

func (c *Client) CreateNode(node *Node) (*Node, error) {
	urlStr := c.buildURL("api", "v1", "nodes")
	body, err := json.Marshal(node)
	if err != nil {
		return nil, fmt.Errorf("marshalling node: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, urlStr, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, decodeAPIError(resp)
	}

	var created Node
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &created, nil
}

func (c *Client) GetNode(name string) (*Node, error) {
	urlStr := c.buildURL("api", "v1", "nodes", name)
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIError(resp)
	}

	var node Node
	if err := json.NewDecoder(resp.Body).Decode(&node); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &node, nil
}

func (c *Client) ListNodes(status NodeStatus) ([]Node, error) {
	urlStr := c.buildURL("api", "v1", "nodes")
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIError(resp)
	}

	var all []Node
	if err := json.NewDecoder(resp.Body).Decode(&all); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	if status == "" {
		return all, nil
	}
	out := make([]Node, 0, len(all))
	for _, n := range all {
		if n.Status == status {
			out = append(out, n)
		}
	}
	return out, nil
}

func (c *Client) UpdateNode(node *Node) error {
	if node == nil || node.Name == "" {
		return fmt.Errorf("node name must be specified for update")
	}
	urlStr := c.buildURL("api", "v1", "nodes", node.Name)

	body, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("marshalling node: %w", err)
	}
	req, err := http.NewRequest(http.MethodPut, urlStr, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decodeAPIError(resp)
	}
	return nil
}
