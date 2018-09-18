package htcondor

import (
	"fmt"
	"os/exec"

	"github.com/retzkek/htcondor-go/classad"
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
}

// NewCommand creates a new HTCondor command.
func NewCommand(command string) *Command {
	return &Command{
		Command: command,
	}
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
		args = append(args, "-af:lrng")
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

// Run runs the command and returns the ClassAds.
// Use Cmd() if you need more control over the handling of the output.
func (c *Command) Run() ([]classad.ClassAd, error) {
	cmd := c.Cmd()
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	ads, err := classad.ReadClassAds(out)
	if err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}
	return ads, nil
}
