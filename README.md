# Condorbeat

Condorbeat is a tool to collect raw ClassAds from HTCondor on a
defined period and publish them to a variety of outputs, including
Kafka, Logstash, and Elasticsearch. Supported ClassAds include:

* Curent jobs in the queue (`condor_q`)
* Completed jobs (`condor_history`)
* Machine, daemons, etc (`condor_status`)

## Running

The fastest way to get condorbeat running is with Docker, using the
[retzkek/condorbeat](https://hub.docker.com/r/retzkek/condorbeat/)
image off Docker Hub and bind-mount in your configuration file:

    docker pull retzkek/condorbeat
	docker run -v /path/to/condorbeat.yml:/condorbeat/condorbeat.yml retzkek/condorbeat

See instructions below for building the binary from source otherwise.

## Configuration

Condorbeat-specific options are under the `condorbeat` section. For
all other options see the [Beats
documentation](https://www.elastic.co/guide/en/beats/libbeat/6.3/beats-reference.html)

* `period` time between ClassAd collection in Go [time.Duration
  format](https://golang.org/pkg/time/#ParseDuration) (default `68s`).
* `pool` HTCondor collector/central manager to query. Leave blank for
  local machine (default).
* `checkpoint_file` name of file to store checkpoints (last data
  collection) under `data` directory (default `checkpoints`).

### Queue

Current jobs in the queue (`condor_q`).

* `classads` collect job classads (default `true`).

### History

Collect completed jobs (`condor_history`). *Note that `condor_history`
is not very efficient at collecting "jobs completed since time t" so
it's preferable to use `filebeat` directly on the schedd to collect
the history spool file in realtime.*

* `classads` collect job classads (default `true`).
* `limit` maximum number of job histories to collect during each
  period.

### Status

Machine, daemon, and any other classads (`condor_status`). Provide a
  list of ad types to collect. Default:

    - type: Collector
	- type: Scheduler
	- type: Negotiator

* `type` daemon/classAd type (`MyType`).
* `constraint` optional constraint to apply

## Building

To build a stand-alone executable for your current platform run `make`.

TODO: upgrade to libbeat 6.5+ to take advantage of the better
cross-platform release build.
