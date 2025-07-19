package core

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/BogdanDolia/ops-butler/new-ops-butler/internal/models"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"go.uber.org/zap"
)

// K8sConfig represents Kubernetes configuration
type K8sConfig struct {
	KubeConfig     string `json:"kubeconfig"`      // Path to kubeconfig file
	InCluster      bool   `json:"in_cluster"`      // Whether to use in-cluster config
	Namespace      string `json:"namespace"`       // Namespace to create jobs in
	JobTTLSeconds  int32  `json:"job_ttl_seconds"` // TTL for completed jobs
	JobImage       string `json:"job_image"`       // Image to use for jobs
	JobPullPolicy  string `json:"job_pull_policy"` // Image pull policy for jobs
	JobServiceAccount string `json:"job_service_account"` // Service account for jobs
}

// K8sClient represents a Kubernetes client
type K8sClient struct {
	clientset *kubernetes.Clientset
	config    *K8sConfig
	logger    *zap.Logger
}

// NewK8sClient creates a new Kubernetes client
func NewK8sClient(cfg *K8sConfig, logger *zap.Logger) (*K8sClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kubernetes configuration is required")
	}

	var config *rest.Config
	var err error

	if cfg.InCluster {
		// Use in-cluster config
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from flags: %w", err)
		}
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &K8sClient{
		clientset: clientset,
		config:    cfg,
		logger:    logger,
	}, nil
}

// CreateJobForTask creates a Kubernetes Job for a task
func (c *K8sClient) CreateJobForTask(task *models.TaskInstance) error {
	if task == nil {
		return fmt.Errorf("task is required")
	}

	// Generate job name
	jobName := fmt.Sprintf("ops-butler-task-%d-%d", task.ID, time.Now().Unix())

	// Create job spec
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: c.config.Namespace,
			Labels: map[string]string{
				"app":      "ops-butler",
				"task-id":  fmt.Sprintf("%d", task.ID),
				"task-type": string(task.TaskType),
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: &c.config.JobTTLSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":      "ops-butler",
						"task-id":  fmt.Sprintf("%d", task.ID),
						"task-type": string(task.TaskType),
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "task",
							Image:           c.config.JobImage,
							ImagePullPolicy: corev1.PullPolicy(c.config.JobPullPolicy),
							Env: []corev1.EnvVar{
								{
									Name:  "TASK_ID",
									Value: fmt.Sprintf("%d", task.ID),
								},
								{
									Name:  "TASK_TYPE",
									Value: string(task.TaskType),
								},
							},
						},
					},
				},
			},
		},
	}

	// Add service account if specified
	if c.config.JobServiceAccount != "" {
		job.Spec.Template.Spec.ServiceAccountName = c.config.JobServiceAccount
	}

	// Add task parameters as environment variables
	if task.Params != nil {
		paramsJSON, err := json.Marshal(task.Params)
		if err != nil {
			return fmt.Errorf("failed to marshal task parameters: %w", err)
		}
		job.Spec.Template.Spec.Containers[0].Env = append(
			job.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  "TASK_PARAMS",
				Value: string(paramsJSON),
			},
		)
	}

	// Add script if task has a template
	if task.Template != nil && task.Template.Script != "" {
		job.Spec.Template.Spec.Containers[0].Env = append(
			job.Spec.Template.Spec.Containers[0].Env,
			corev1.EnvVar{
				Name:  "TASK_SCRIPT",
				Value: task.Template.Script,
			},
		)
	}

	// Create job
	createdJob, err := c.clientset.BatchV1().Jobs(c.config.Namespace).Create(
		context.Background(),
		job,
		metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	// Update task with job name
	task.JobName = createdJob.Name
	task.State = models.TaskStateScheduled

	c.logger.Info("Created job for task",
		zap.Uint("task_id", task.ID),
		zap.String("job_name", createdJob.Name),
		zap.String("namespace", c.config.Namespace),
	)

	return nil
}

// GetJobStatus gets the status of a job
func (c *K8sClient) GetJobStatus(jobName string) (models.TaskState, int, error) {
	job, err := c.clientset.BatchV1().Jobs(c.config.Namespace).Get(
		context.Background(),
		jobName,
		metav1.GetOptions{},
	)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get job: %w", err)
	}

	// Determine job status
	if job.Status.Active > 0 {
		return models.TaskStateRunning, 0, nil
	} else if job.Status.Succeeded > 0 {
		return models.TaskStateCompleted, 0, nil
	} else if job.Status.Failed > 0 {
		return models.TaskStateFailed, 1, nil
	}

	return models.TaskStateScheduled, 0, nil
}

// DeleteJob deletes a job
func (c *K8sClient) DeleteJob(jobName string) error {
	propagationPolicy := metav1.DeletePropagationBackground
	return c.clientset.BatchV1().Jobs(c.config.Namespace).Delete(
		context.Background(),
		jobName,
		metav1.DeleteOptions{
			PropagationPolicy: &propagationPolicy,
		},
	)
}

// GetJobLogs gets the logs from a job
func (c *K8sClient) GetJobLogs(jobName string) (string, error) {
	// Get pods for job
	pods, err := c.clientset.CoreV1().Pods(c.config.Namespace).List(
		context.Background(),
		metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", jobName),
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to list pods for job: %w", err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job")
	}

	// Get logs from first pod
	podName := pods.Items[0].Name
	logs, err := c.clientset.CoreV1().Pods(c.config.Namespace).GetLogs(
		podName,
		&corev1.PodLogOptions{},
	).Do(context.Background()).Raw()
	if err != nil {
		return "", fmt.Errorf("failed to get logs from pod: %w", err)
	}

	return string(logs), nil
}