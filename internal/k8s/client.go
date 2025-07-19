package k8s

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"go.uber.org/zap"
)

// Client represents a Kubernetes client for interacting with the cluster
type Client struct {
	logger *zap.Logger
}

// NewClient creates a new Kubernetes client
func NewClient(logger *zap.Logger) *Client {
	return &Client{
		logger: logger,
	}
}

// GetPodLogs retrieves logs from a specific pod in a namespace
func (c *Client) GetPodLogs(ctx context.Context, namespace, podName string, opts *LogOptions) (string, error) {
	if opts == nil {
		opts = &LogOptions{
			Lines:    100,
			Follow:   false,
			Previous: false,
		}
	}

	// Build kubectl command
	args := []string{"logs", podName, "-n", namespace}

	if opts.Container != "" {
		args = append(args, "-c", opts.Container)
	}

	if opts.Lines > 0 {
		args = append(args, "--tail", fmt.Sprintf("%d", opts.Lines))
	}

	if opts.Previous {
		args = append(args, "--previous")
	}

	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	// Execute kubectl command
	cmd := exec.CommandContext(ctx, "kubectl", args...)

	c.logger.Info("Executing kubectl command",
		zap.String("command", "kubectl "+strings.Join(args, " ")))

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.Error("Failed to get pod logs",
			zap.Error(err),
			zap.String("namespace", namespace),
			zap.String("pod", podName),
			zap.String("output", string(output)))
		return "", fmt.Errorf("failed to get logs for pod %s in namespace %s: %w", podName, namespace, err)
	}

	return string(output), nil
}

// ListPods retrieves a list of pods in a namespace
func (c *Client) ListPods(ctx context.Context, namespace string, labelSelector string) ([]string, error) {
	args := []string{"get", "pods", "-n", namespace, "--no-headers", "-o", "custom-columns=:metadata.name"}

	if labelSelector != "" {
		args = append(args, "-l", labelSelector)
	}

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	c.logger.Info("Executing kubectl command",
		zap.String("command", "kubectl "+strings.Join(args, " ")))

	output, err := cmd.CombinedOutput()
	if err != nil {
		c.logger.Error("Failed to list pods",
			zap.Error(err),
			zap.String("namespace", namespace),
			zap.String("output", string(output)))
		return nil, fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	pods := make([]string, 0, len(lines))

	for _, line := range lines {
		pod := strings.TrimSpace(line)
		if pod != "" {
			pods = append(pods, pod)
		}
	}

	return pods, nil
}

// CheckPodExists checks if a pod exists in a namespace
func (c *Client) CheckPodExists(ctx context.Context, namespace, podName string) (bool, error) {
	args := []string{"get", "pod", podName, "-n", namespace, "--no-headers"}

	cmd := exec.CommandContext(ctx, "kubectl", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// If the command fails, the pod likely doesn't exist
		if strings.Contains(string(output), "not found") {
			return false, nil
		}

		c.logger.Error("Failed to check pod existence",
			zap.Error(err),
			zap.String("namespace", namespace),
			zap.String("pod", podName),
			zap.String("output", string(output)))
		return false, fmt.Errorf("failed to check pod %s in namespace %s: %w", podName, namespace, err)
	}

	return true, nil
}

// LogOptions represents options for retrieving pod logs
type LogOptions struct {
	Container string // Specific container name (optional)
	Lines     int    // Number of lines to retrieve (0 for all)
	Follow    bool   // Whether to follow logs (stream)
	Previous  bool   // Get logs from previous container instance
	Since     string // Time duration to get logs since (e.g., "1h", "5m")
}
