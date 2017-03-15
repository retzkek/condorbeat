// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period  time.Duration `config:"period"`
	Pool    string        `config:"pool"`
	Queue   queueConfig   `config:"queue"`
	History historyConfig `config:"history"`
	Status  statusConfig  `config:"status"`
}

type queueConfig struct {
	Classads bool `config:"classads"`
	Metrics  bool `config:"metrics"`
}

type historyConfig struct {
	Classads       bool   `config:"classads"`
	Metrics        bool   `config:"metrics"`
	Limit          int    `config:"limit"`
	CheckpointFile string `config:"checkpoint_file"`
}

type statusConfig struct {
	Collector      bool `config:"collector"`
	Schedd         bool `config:"schedd"`
	Negotiator     bool `config:"negotiator"`
	Startd         bool `config:"startd"`
	SlotMetrics    bool `config:"slot_metrics"`
	GlideinMetrics bool `config:"glidein_metrics"`
}

var DefaultConfig = Config{
	Period: 60 * time.Second,
	Pool:   "",
	Queue: queueConfig{
		Classads: true,
		Metrics:  true,
	},
	History: historyConfig{
		Classads:       true,
		Metrics:        true,
		Limit:          500,
		CheckpointFile: "checkpoints",
	},
	Status: statusConfig{
		Collector:      true,
		Schedd:         true,
		Negotiator:     true,
		Startd:         false,
		SlotMetrics:    true,
		GlideinMetrics: false,
	},
}
