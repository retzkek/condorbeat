package htcondor

import (
	"context"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/groupcache"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/retzkek/htcondor-go/classad"
)

var (
	commandDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name: "htcondor_client_command_duration_seconds",
			Help: "Histogram of command runtimes.",
		},
	)
)

const (
	keySeparator    = "\x1f"     // unit separator
	attributeFormat = "-af:lrng" // format command for condor attributes
)

// Command represents an HTCondor command-line tool, e.g. condor_q.
//
// It implements a builder pattern, so you can call e.g.
//     NewCommand("condor_q").WithPool("mypool:9618").WithName("myschedd").WithConstraint("Owner == \"Me\"")
//
// You can also build it directly, e.g.
// c := Command{
//    Command: "condor_q",
//    Pool: "mypool:9618",
//    Name: "myschedd",
//    Constraint: "Owner == \"Me\"",
// }
type Command struct {
	// Command is HTCondor command to run.
	Command string
	// Pool is HTCondor pool (collector) to query.
	Pool string
	// Name is the -name argument.
	Name string
	// Limit is the -limit argument.
	Limit int
	// Constraint sets the -constraint argument.
	Constraint string
	// Attributes is a list of specific attributes to return.
	// If Attributes is empty, all attributes are returned.
	Attributes []string
	// Args is a list of any extra arguments to pass.
	Args []string
	// cache is an optional groupcache pool to cache
	// queries. Inititalize with WithCache().
	cache         *groupcache.HTTPPool
	cacheGroup    string
	cacheLifetime time.Duration
}

// NewCommand creates a new HTCondor command.
func NewCommand(command string) *Command {
	return &Command{
		Command: command,
	}
}

// WithCache initializes a groupcache group for the client. Set cacheLifetime to
// 0 to *never* expire cached queries (unless they are LRU evicted).
func (c *Command) WithCache(pool *groupcache.HTTPPool, group string, cacheBytes int64, cacheLifetime time.Duration) *Command {
	c.cache = pool
	c.cacheGroup = group
	c.cacheLifetime = cacheLifetime
	if groupcache.GetGroup(group) == nil {
		groupcache.NewGroup(c.cacheGroup, cacheBytes, commandGetter())
	}
	return c
}

// WithPool sets the -pool argument for the command.
func (c *Command) WithPool(pool string) *Command {
	c.Pool = pool
	return c
}

// WithName sets the -name argument for the command.
func (c *Command) WithName(name string) *Command {
	c.Name = name
	return c
}

// WithLimit sets the -limit argument for the command.
func (c *Command) WithLimit(limit int) *Command {
	c.Limit = limit
	return c
}

// WithConstraint set the -constraint argument for the command.
func (c *Command) WithConstraint(constraint string) *Command {
	c.Constraint = constraint
	return c
}

// WithAttribute sets a specific attribute to return, rather than the entire
// ClassAd. Can be called multiple times.
func (c *Command) WithAttribute(attribute string) *Command {
	if c.Attributes == nil {
		c.Attributes = []string{attribute}
	} else {
		c.Attributes = append(c.Attributes, attribute)
	}
	return c
}

// WithArg adds an extra argument to pass. Can be called multiple times.
func (c *Command) WithArg(arg string) *Command {
	if c.Args == nil {
		c.Args = []string{arg}
	} else {
		c.Args = append(c.Args, arg)
	}
	return c
}

// MakeArgs builds the complete argument list to be passed to the command.
func (c *Command) MakeArgs() []string {
	args := make([]string, 0)
	if c.Pool != "" {
		args = append(args, "-pool", c.Pool)
	}
	if c.Name != "" {
		args = append(args, "-name", c.Name)
	}
	if c.Limit > 0 {
		args = append(args, "-limit", fmt.Sprintf("%d", c.Limit))
	}
	if c.Constraint != "" {
		args = append(args, "-constraint", c.Constraint)
	}
	if len(c.Args) > 0 {
		args = append(args, c.Args...)
	}
	if len(c.Attributes) > 0 {
		args = append(args, attributeFormat)
		args = append(args, c.Attributes...)
	} else {
		args = append(args, "-long")
	}
	return args
}

// Cmd generates an exec.Cmd you can use to run the command manually.
// Use Run() to run the command and get back ClassAds.
func (c *Command) Cmd() *exec.Cmd {
	return exec.Command(c.Command, c.MakeArgs()...)
}

// encodeKey encodes the command into a string, to be used as a cache key.
func (c *Command) encodeKey() string {
	timeKey := "0"
	if c.cacheLifetime > 0 {
		timeKey = time.Now().Truncate(c.cacheLifetime).Format(time.RFC3339)
	}
	return timeKey + keySeparator +
		c.Command + keySeparator +
		strings.Join(c.MakeArgs(), keySeparator)
}

// decodeKey decodes the command from a key string. It does not restore the
// original Command, instead putting all the arguments into Args.
func decodeKey(key string) (*Command, error) {
	parts := strings.Split(key, keySeparator)
	if len(parts) < 2 {
		return nil, fmt.Errorf("unable to decode cache key: %s", key)
	}
	// first field is time key, we don't need it
	c := Command{
		Command: parts[1],
	}
	if len(parts) > 2 {
		endArgs := len(parts) - 1
		for i, arg := range parts {
			if arg == attributeFormat {
				endArgs = i
				break
			}
		}
		c.Args = parts[2:endArgs]
		if endArgs < len(parts)-1 {
			c.Attributes = parts[endArgs+1:]
		}
	}
	return &c, nil
}

// commandGetter returns a groupCache.GetterFunc that queries HTCondor with the
// configured command, and stores the raw response in dest.
func commandGetter() groupcache.GetterFunc {
	return func(ctx context.Context, key string, dest groupcache.Sink) error {
		timer := prometheus.NewTimer(commandDuration)
		defer timer.ObserveDuration()

		c, err := decodeKey(key)
		if err != nil {
			return err
		}
		cmd := c.Cmd()
		out, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		if err := cmd.Start(); err != nil {
			return err
		}
		resp, err := ioutil.ReadAll(out)
		if err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return err
		}
		return dest.SetBytes(resp)
	}
}

// Run runs the command and returns the ClassAds.
// Use Cmd() if you need more control over the handling of the output.
func (c *Command) Run() ([]classad.ClassAd, error) {
	key := c.encodeKey()
	var resp groupcache.ByteView
	var err error
	if c.cache != nil {
		group := groupcache.GetGroup(c.cacheGroup)
		err = group.Get(nil, key, groupcache.ByteViewSink(&resp))
	} else {
		// call the getter directly
		err = commandGetter()(nil, key, groupcache.ByteViewSink(&resp))
	}
	if err != nil {
		return nil, err
	}
	ads, err := classad.ReadClassAds(resp.Reader())
	if err != nil {
		return nil, err
	}
	return ads, nil
}

// Stream runs the command and sends the ClassAds on a channel. Errors are
// returned on a separate channel. Both will be closed when the command is done.
//
// N.B. if using Stream with a cache you'll lose much of performance and memory
// advantages of streaming, since the entire HTCondor response must be read,
// whether from HTCondor or from the cache, before the classads can be sent.
func (c *Command) Stream(ch chan classad.ClassAd, errors chan error) {
	if c.cache != nil {
		key := c.encodeKey()
		var resp groupcache.ByteView
		var err error
		group := groupcache.GetGroup(c.cacheGroup)
		err = group.Get(nil, key, groupcache.ByteViewSink(&resp))
		if err != nil {
			errors <- fmt.Errorf("error getting response from cache: %s", err)
			close(errors)
			close(ch)
			return
		}
		classad.StreamClassAds(resp.Reader(), ch, errors)
	} else {
		cmd := c.Cmd()
		out, err := cmd.StdoutPipe()
		if err != nil {
			errors <- fmt.Errorf("error opening command pipe: %s", err)
			close(errors)
			close(ch)
			return
		}
		if err := cmd.Start(); err != nil {
			errors <- fmt.Errorf("error running command: %s", err)
			errors <- err
			close(errors)
			close(ch)
			return
		}
		classad.StreamClassAds(out, ch, errors)
		cmd.Wait()
	}
}
