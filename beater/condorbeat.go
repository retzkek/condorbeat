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
	checkpoints        map[string]common.Time
}

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Condorbeat{
		done:               make(chan struct{}),
		config:             config,
		checkpointFilePath: b.Config.Path.Resolve(paths.Data, config.History.CheckpointFile),
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

	// launch collectors
	eventch := make(chan common.MapStr)
	stopchs := make([]*chan bool, len(schedds))
	for i, schedd := range schedds {
		id := base64.StdEncoding.EncodeToString([]byte("condor_history-" + schedd))
		cmd := &htcondor.Command{
			Command: "condor_history",
			Pool:    bt.config.Pool,
			Name:    schedd,
			Limit:   bt.config.History.Limit,
			Args:    []string{"-forwards"},
		}
		stopch := make(chan bool)
		stopchs[i] = &stopch
		go runPeriodicCommand(id, cmd, bt.checkpoints[id], bt.config.Period, eventch, stopch)
	}

	// publish events
	bt.client = b.Publisher.Connect()
	updateTicker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-bt.done:
			logp.Debug("beater", "stopping collectors...")
			for _, ch := range stopchs {
				*ch <- true
			}
			logp.Debug("beater", "exiting")
			return nil
		case <-updateTicker.C:
			logp.Info("saving checkpoints to file %s", bt.checkpointFilePath)
			err := bt.saveCheckpoints(bt.checkpointFilePath)
			if err != nil {
				logp.Err("error saving checkpoints: %s", err)
			}
		case event := <-eventch:
			logp.Debug("beater", "%v", event)
			bt.client.PublishEvent(event, publisher.Sync)
			logp.Info("Event sent")
			bt.checkpoints[event["beat"].(common.MapStr)["collector_id"].(string)] = event["@timestamp"].(common.Time)
			logp.Debug("beater", "checkpoints: %s", bt.checkpointString())
		}
	}
}

func (bt *Condorbeat) Stop() {
	bt.client.Close()
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

func runPeriodicCommand(id string, cmd *htcondor.Command, checkpoint common.Time, period time.Duration, events chan common.MapStr, stop chan bool) {
	ticker := time.NewTicker(period)
	baseConstraint := cmd.Constraint
	for {
		logp.Debug("collector", "waiting...")
		select {
		case <-stop:
			logp.Debug("collector", "exiting")
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
		for _, ad := range ads {
			endtime := common.Time(time.Unix(ad["EnteredCurrentStatus"].Value.(int64), 0))
			event := common.MapStr{
				"@timestamp": endtime,
				"type":       "condor_history",
				"beat":       common.MapStr{"collector_id": id},
			}
			for k, v := range ad {
				event[k] = v.Value
			}
			logp.Debug("collector", "sending event")
			events <- event
			if time.Time(endtime).After(time.Time(checkpoint)) {
				checkpoint = endtime
			}
		}
	}
}
