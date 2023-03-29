// Config is put into a different package to prevent cyclic imports in case
// it is needed in several locations

package config

import "time"

type Config struct {
	Period           time.Duration  `config:"period"`
	Pool             string         `config:"pool"`
	ScheddConstraint string         `config:"schedd_constraint"`
	CheckpointFile   string         `config:"checkpoint_file"`
	Queue            queueConfig    `config:"queue"`
	History          historyConfig  `config:"history"`
	Status           []statusConfig `config:"status"`
}

type queueConfig struct {
	Classads   bool   `config:"classads"`
	Constraint string `config:"constraint"`
}

type historyConfig struct {
	Classads bool `config:"classads"`
	Limit    int  `config:"limit"`
}

type statusConfig struct {
	MyType     string `config:"type"`
	Constraint string `config:"constraint"`
}

var DefaultConfig = Config{
	Period:         60 * time.Second,
	Pool:           "",
	CheckpointFile: "checkpoints",
	Queue: queueConfig{
		Classads:   true,
		Constraint: "",
	},
	History: historyConfig{
		Classads: true,
		Limit:    10000,
	},
	Status: []statusConfig{
		{MyType: "Collector"},
		{MyType: "Scheduler"},
		{MyType: "Negotiator"},
	},
}
