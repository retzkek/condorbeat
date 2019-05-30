package beater

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/paths"

	"github.com/retzkek/condorbeat/config"
	htcondor "github.com/retzkek/htcondor-go"
	"github.com/retzkek/htcondor-go/classad"
)

type Condorbeat struct {
	done               chan struct{}
	config             config.Config
	client             beat.Client
	checkpointFilePath string
	checkpoints        Checkpoint
}

type Checkpoint map[string]time.Time

// Creates beater
func New(b *beat.Beat, cfg *common.Config) (beat.Beater, error) {
	config := config.DefaultConfig
	if err := cfg.Unpack(&config); err != nil {
		return nil, fmt.Errorf("Error reading config file: %v", err)
	}

	bt := &Condorbeat{
		done:               make(chan struct{}),
		config:             config,
		checkpointFilePath: paths.Resolve(paths.Data, config.CheckpointFile),
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

	var err error
	bt.client, err = b.Publisher.Connect()
	if err != nil {
		return err
	}
	defer bt.client.Close()

	checkch := make(chan Checkpoint)
	var wg sync.WaitGroup
	// launch queue collectors
	if bt.config.Queue.Classads {
		for _, schedd := range schedds {
			wg.Add(1)
			go bt.collectQueue(bt.config.Pool, schedd, bt.config.Period, &wg)
		}
	}
	// launch history collectors
	if bt.config.History.Classads {
		for _, schedd := range schedds {
			wg.Add(1)
			id := "condor_history-" + bt.config.Pool + "-" + schedd
			cmd := &htcondor.Command{
				Command: "condor_history",
				Pool:    bt.config.Pool,
				Name:    schedd,
				Limit:   bt.config.History.Limit,
				//Args:    []string{"-forwards"},
			}
			go bt.collectHistory(id, cmd, bt.checkpoints[id], bt.config.Period, checkch, &wg)
		}
	}
	// launch status collectors
	for _, c := range bt.config.Status {
		wg.Add(1)
		go bt.collectStatus(bt.config.Pool, c.MyType, c.Constraint, bt.config.Period, &wg)
	}

	// wait loop
	for {
		select {
		case <-bt.done:
			logp.Debug("beater", "waiting for collectors to stop")
			wg.Wait()
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
		bt.checkpoints = make(map[string]time.Time)
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
	for name, c := range bt.checkpoints {
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

func (bt *Condorbeat) collectQueue(pool, name string, period time.Duration, done *sync.WaitGroup) {
	defer done.Done()
	id := "condor_q-" + pool + "-" + name
	cmd := &htcondor.Command{
		Command: "condor_q",
		Pool:    pool,
		Name:    name,
		Args:    []string{"-allusers"},
	}
	ticker := time.NewTicker(period)
	for {
		logp.Debug("collector", "running command %s with args %v", cmd.Command, cmd.MakeArgs())
		ads := make(chan classad.ClassAd)
		errors := make(chan error)
		go cmd.Stream(ads, errors)
		for {
			select {
			case ad, ok := <-ads:
				if !ok {
					ads = nil
					continue
				}
				event := beat.Event{
					Timestamp: time.Now(),
					Fields: common.MapStr{
						"type":         "job",
						"collector_id": id,
					},
				}
				for k, v := range ad {
					event.Fields[k] = v.Value
				}
				bt.client.Publish(event)
			case err, ok := <-errors:
				if !ok {
					errors = nil
					continue
				}
				logp.Err("error running condor command %s: %s", cmd.Command, err)
			default:
			}
			if ads == nil && errors == nil {
				break
			}
		}

		logp.Debug("collector", "%s sleeping %s...", id, period.String())
		select {
		case <-bt.done:
			logp.Debug("collector", "%s exiting", id)
			return
		case <-ticker.C:
		}
	}
}

func (bt *Condorbeat) collectHistory(id string, cmd *htcondor.Command, checkpoint time.Time, period time.Duration, check chan Checkpoint, done *sync.WaitGroup) {
	defer done.Done()
	ticker := time.NewTicker(period)
	baseConstraint := cmd.Constraint
	var newCheckpoint time.Time
	for {
		if baseConstraint == "" {
			cmd.Constraint = fmt.Sprintf("EnteredCurrentStatus > %d", checkpoint.Unix())
		} else {
			cmd.Constraint = fmt.Sprintf("%s && (EnteredCurrentStatus > %d)", baseConstraint, checkpoint.Unix())
		}
		logp.Debug("collector", "running command %s with args %v", cmd.Command, cmd.MakeArgs())
		ads := make(chan classad.ClassAd)
		errors := make(chan error)
		go cmd.Stream(ads, errors)
		for {
			select {
			case ad, ok := <-ads:
				if !ok {
					ads = nil
					continue
				}

				endtime := time.Unix(ad["EnteredCurrentStatus"].Value.(int64), 0)

				if endtime.After(checkpoint) {
					newCheckpoint = endtime
				}
				event := beat.Event{
					Timestamp: time.Now(),
					Fields: common.MapStr{
						"type":         "job",
						"collector_id": id,
					},
				}
				for k, v := range ad {
					event.Fields[k] = v.Value
				}
				bt.client.Publish(event)
				checkpoint = newCheckpoint
				check <- Checkpoint{id: checkpoint}
			case err, ok := <-errors:
				if !ok {
					errors = nil
					continue
				}
				logp.Err("error running condor command %s: %s", cmd.Command, err)
			default:
			}
			if ads == nil && errors == nil {
				break
			}
		}

		logp.Debug("collector", "%s sleeping %s...", id, period.String())
		select {
		case <-bt.done:
			logp.Debug("collector", "%s exiting", id)
			return
		case <-ticker.C:
		}
	}
}

func (bt *Condorbeat) collectStatus(pool, daemonType string, constraint string, period time.Duration, done *sync.WaitGroup) {
	defer done.Done()
	id := "condor_status-" + pool + "-" + daemonType
	myType := fmt.Sprintf("%s_status", daemonType)
	cons := fmt.Sprintf("MyType==\"%s\"", daemonType)
	if constraint != "" {
		cons = fmt.Sprintf("(%s)&&(%s)", cons, constraint)
	}
	cmd := &htcondor.Command{
		Command:    "condor_status",
		Pool:       pool,
		Args:       []string{"-any"},
		Constraint: cons,
	}
	ticker := time.NewTicker(period)
	for {
		logp.Debug("collector", "running command %s with args %v", cmd.Command, cmd.MakeArgs())
		ads := make(chan classad.ClassAd)
		errors := make(chan error)
		go cmd.Stream(ads, errors)
		for {
			select {
			case ad, ok := <-ads:
				if !ok {
					ads = nil
					continue
				}
				event := beat.Event{
					Timestamp: time.Now(),
					Fields: common.MapStr{
						"type":         "job",
						"collector_id": id,
					},
				}
				for k, v := range ad {
					event.Fields[k] = v.Value
				}
				bt.client.Publish(event)
			case err, ok := <-errors:
				if !ok {
					errors = nil
					continue
				}
				logp.Err("error running condor command %s: %s", cmd.Command, err)
			default:
			}
			if ads == nil && errors == nil {
				break
			}
		}

		logp.Debug("collector", "%s sleeping %s...", id, period.String())
		select {
		case <-bt.done:
			logp.Debug("collector", "%s exiting", id)
			return
		case <-ticker.C:
		}
	}
}
