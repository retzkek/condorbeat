package beater

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"
	"github.com/elastic/beats/libbeat/publisher"

	"github.com/retzkek/condorbeat/config"
	htcondor "github.com/retzkek/htcondor-go"
)

type Condorbeat struct {
	done               chan struct{}
	config             config.Config
	client             publisher.Client
	checkpointFilePath string
	checkpoints        Checkpoint
}

type Checkpoint map[string]common.Time

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Condorbeat{
		done:               make(chan struct{}),
		config:             config,
		checkpointFilePath: b.Config.Path.Resolve(paths.Data, config.CheckpointFile),
	}

	logp.Info("loading checkpoints from file %s", bt.checkpointFilePath)
	if err := bt.loadCheckpoints(bt.checkpointFilePath); err != nil {
		return nil, fmt.Errorf("Error reading/creating checkpoints file: %v", err)
	}
	logp.Info("checkpoints: %s", bt.checkpointString())
	return bt, nil
}

func (bt *Condorbeat) Run(b *beat.Beat) error {
	logp.Info("condorbeat is running! Hit CTRL-C to stop it.")

	schedds := []string{""}
	if bt.config.Pool != "" {
		var err error
		schedds, err = getSchedds(bt.config.Pool)
		if err != nil {
			return err
		}
	}

	checkch := make(chan Checkpoint)
	stopchs := make([]*chan bool, 0)
	// launch queue collectors
	if bt.config.Queue.Classads || bt.config.Queue.Metrics {
		for _, schedd := range schedds {
			stopch := make(chan bool)
			stopchs = append(stopchs, &stopch)
			go collectQueue(bt.config.Pool, schedd, bt.config.Period, b.Publisher.Connect(), stopch)
		}
	}
	// launch history collectors
	if bt.config.History.Classads || bt.config.History.Metrics {
		for _, schedd := range schedds {
			id := base64.StdEncoding.EncodeToString([]byte("condor_history-" + schedd))
			cmd := &htcondor.Command{
				Command: "condor_history",
				Pool:    bt.config.Pool,
				Name:    schedd,
				Limit:   bt.config.History.Limit,
				Args:    []string{"-forwards"},
			}
			stopch := make(chan bool)
			stopchs = append(stopchs, &stopch)
			go collectHistory(bt.config.Pool, cmd, bt.checkpoints[id], bt.config.Period, b.Publisher.Connect(), checkch, stopch)
		}
	}
	// launch status collectors
	if bt.config.Status.Collector {
		stopch := make(chan bool)
		stopchs = append(stopchs, &stopch)
		go collectStatus(bt.config.Pool, "collector", bt.config.Period, b.Publisher.Connect(), stopch)
	}
	if bt.config.Status.Schedd {
		stopch := make(chan bool)
		stopchs = append(stopchs, &stopch)
		go collectStatus(bt.config.Pool, "schedd", bt.config.Period, b.Publisher.Connect(), stopch)
	}
	if bt.config.Status.Negotiator {
		stopch := make(chan bool)
		stopchs = append(stopchs, &stopch)
		go collectStatus(bt.config.Pool, "negotiator", bt.config.Period, b.Publisher.Connect(), stopch)
	}
	if bt.config.Status.Startd {
		stopch := make(chan bool)
		stopchs = append(stopchs, &stopch)
		go collectStatus(bt.config.Pool, "startd", bt.config.Period, b.Publisher.Connect(), stopch)
	}

	// wait loop
	for {
		select {
		case <-bt.done:
			logp.Debug("beater", "stopping collectors...")
			for _, ch := range stopchs {
				*ch <- true
			}
			logp.Debug("beater", "exiting")
			return nil
		case checkpoint := <-checkch:
			for k, v := range checkpoint {
				bt.checkpoints[k] = v
			}
			logp.Debug("beater", "checkpoints: %s", bt.checkpointString())
			logp.Info("saving checkpoints to file %s", bt.checkpointFilePath)
			err := bt.saveCheckpoints(bt.checkpointFilePath)
			if err != nil {
				logp.Err("error saving checkpoints: %s", err)
			}
		}
	}
}

func (bt *Condorbeat) Stop() {
	close(bt.done)
}

func (bt *Condorbeat) loadCheckpoints(filePath string) error {
	file, err := os.Open(filePath)
	defer file.Close()
	if os.IsNotExist(err) {
		bt.checkpoints = make(map[string]common.Time)
		return nil
	} else if err != nil {
		return err
	}
	dec := json.NewDecoder(file)
	return dec.Decode(&bt.checkpoints)
}

func (bt *Condorbeat) saveCheckpoints(filePath string) error {
	file, err := os.Create(filePath)
	defer file.Close()
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	return enc.Encode(bt.checkpoints)
}

func (bt *Condorbeat) checkpointString() string {
	str := "{"
	for k, c := range bt.checkpoints {
		name, err := base64.StdEncoding.DecodeString(k)
		if err != nil {
			name = []byte(k)
		}
		str += string(name) + ":" + time.Time(c).String() + ", "
	}
	str += "}"
	return str
}

func getSchedds(pool string) ([]string, error) {
	ads, err := htcondor.NewCommand("condor_status").WithPool(pool).WithArg("-schedd").WithAttribute("name").Run()
	if err != nil {
		return nil, err
	}
	schedds := make([]string, len(ads))
	for i, ad := range ads {
		schedds[i] = ad["name"].String()
	}
	return schedds, nil
}

func collectQueue(pool, name string, period time.Duration, client publisher.Client, stop chan bool) {
	defer client.Close()
	id := "condor_q-" + pool + "-" + name
	cmd := &htcondor.Command{
		Command: "condor_q",
		Pool:    pool,
		Name:    name,
	}
	ticker := time.NewTicker(period)
	for {
		logp.Debug("collector", "%s sleeping %s...", id, period.String())
		select {
		case <-stop:
			logp.Debug("collector", "%s exiting", id)
			return
		case <-ticker.C:
		}
		logp.Debug("collector", "running command %s with args %v", cmd.Command, cmd.MakeArgs())
		ads, err := cmd.Run()
		if err != nil {
			logp.Err("error running condor command %s: %s", cmd.Command, err)
			continue // just retry next tick
		}
		events := make([]common.MapStr, len(ads))
		for i, ad := range ads {
			event := common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"type":       "job",
				"beat":       common.MapStr{"collector_id": id},
			}
			for k, v := range ad {
				event[k] = v.Value
			}
			events[i] = event
		}
		logp.Debug("collector", "%s publishing events", id)
		client.PublishEvents(events)
	}
}

func collectHistory(id string, cmd *htcondor.Command, checkpoint common.Time, period time.Duration, client publisher.Client, check chan Checkpoint, stop chan bool) {
	defer client.Close()
	ticker := time.NewTicker(period)
	baseConstraint := cmd.Constraint
	var newCheckpoint common.Time
	for {
		logp.Debug("collector", "%s sleeping %s...", id, period.String())
		select {
		case <-stop:
			logp.Debug("collector", "%s exiting", id)
			return
		case <-ticker.C:
		}
		if baseConstraint == "" {
			cmd.Constraint = fmt.Sprintf("EnteredCurrentStatus > %d", time.Time(checkpoint).Unix())
		} else {
			cmd.Constraint = fmt.Sprintf("%s && (EnteredCurrentStatus > %d)", baseConstraint, time.Time(checkpoint).Unix())
		}
		logp.Debug("collector", "running command %s with args %v", cmd.Command, cmd.MakeArgs())
		ads, err := cmd.Run()
		if err != nil {
			logp.Err("error running condor command %s: %s", cmd.Command, err)
			continue // just retry next tick
		}
		events := make([]common.MapStr, len(ads))
		for i, ad := range ads {
			endtime := common.Time(time.Unix(ad["EnteredCurrentStatus"].Value.(int64), 0))
			event := common.MapStr{
				"@timestamp": endtime,
				"type":       "job",
				"beat":       common.MapStr{"collector_id": id},
			}
			for k, v := range ad {
				event[k] = v.Value
			}
			client.PublishEvent(event)
			if time.Time(endtime).After(time.Time(checkpoint)) {
				newCheckpoint = endtime
			}
			events[i] = event
		}
		logp.Debug("collector", "%s publishing events", id)
		ok := client.PublishEvents(events, publisher.Guaranteed, publisher.Sync)
		if !ok {
			return
		}
		checkpoint = newCheckpoint
		check <- Checkpoint{id: checkpoint}
	}
}

func collectStatus(pool, daemonType string, period time.Duration, client publisher.Client, stop chan bool) {
	defer client.Close()
	id := "condor_status-" + pool + "-" + daemonType
	cmd := &htcondor.Command{
		Command: "condor_status",
		Pool:    pool,
		Args:    []string{"-" + daemonType},
	}
	ticker := time.NewTicker(period)
	for {
		logp.Debug("collector", "%s sleeping %s...", id, period.String())
		select {
		case <-stop:
			logp.Debug("collector", "%s exiting", id)
			return
		case <-ticker.C:
		}
		logp.Debug("collector", "running command %s with args %v", cmd.Command, cmd.MakeArgs())
		ads, err := cmd.Run()
		if err != nil {
			logp.Err("error running condor command %s: %s", cmd.Command, err)
			continue // just retry next tick
		}
		events := make([]common.MapStr, len(ads))
		for i, ad := range ads {
			event := common.MapStr{
				"@timestamp": common.Time(time.Now()),
				"type":       "status",
				"beat":       common.MapStr{"collector_id": id},
			}
			for k, v := range ad {
				event[k] = v.Value
			}
			events[i] = event
		}
		logp.Debug("collector", "%s publishing events", id)
		client.PublishEvents(events)
	}
}
