package models

type CGroupMetric struct {
	Name      string
	MountPath string
}

type MetricsPayload struct {
	Timestamp   int64                       `json:"timestamp"`
	ContainerID string                      `json:"container_id"`
	SampleRate  string                      `json:"sample_rate"`
	Metrics     map[string]map[string]int64 `json:"metrics"`
}
