[![Code Climate](https://codeclimate.com/github/therealbill/redskull/badges/gpa.svg)](https://codeclimate.com/github/therealbill/redskull)
[![Stories in Ready](https://badge.waffle.io/therealbill/redskull.png?label=ready&title=Ready)](https://waffle.io/therealbill/redskull)
# What Is Red Skull?

Red Skull is a Sentinel management system. It is designed to run on each
sentinel node you operate and provide a single, yet distributed,
mechanism for managing Sentinel as well as interacting with it.

# Overview

Written in Go, Red Skull runs on each Sentinel and bootstraps itself
from that Sentinel's configuration file. It will then interrogate any
`known-sentinel` directives as well as run `setinel sentinels <name> for
each pod found in the config file.  It essentially crawls through your
Sentinel constellation and discovers all sentinels, masters, and slaves.

It then provides a decent web interface for viewing and managing your
sentinels, and by proxy the Redis instances under management. It
introduces some new concepts/terminology and these will be explained in
the documentation tree.

In addition to the front end Red Skull provides an HTTP/JSON REST-*like*
interface for interacting with programmaticly. Adding the redis Sentinel
API as another interface is planned as well.

# Better Documentation

You can find better guides to redskull at [Redskull.IO](http://redskull.io).



# Current State

Can you use it for "production use". Yes. Will it destroy your setup?
Not likely.  Some of the truly destructive things are disabled, even. Yup, a
bit paranoid sometimes. :)

Most of the things you can do in the web UI are also available in the
JSON+HTTP API but there may be some new functionality I've not yet added
to the API.

## Refactor Update

Ultimately I was displeased with certain aspects of the system and newer
services have come out which can offload some of the distributed systems type
code. This I am currently refactoring Redskull into two main components: an
"agent" which runs on the Sentinel nodes, and a "controller" which you can run
anywhere. Tying these together will be Hashicorp's Consul and eventually
Vault as an optional integration.

The way it will work is that the `redskull-agent` piece will run on
Sentinels and serve to load known pods into consul, update Consul with
information as they change, and be an RPC service for the controller to
fetch auth informtion for pods.

The controller will be a process which runs wherever you want (and as
many as needed for load balancing and availability). It will keep
theexisting functionality of being a front-end and API server. It is
where the "business logic" of running Sentinel+Redis pods will live.

This controller refactor will be taking place on a dedicated branch.
This will allow the existing code to keep working as before until it is
ready for the switch. However, the name has changed to
`redskull-controller` to reflect both the future and recognition of what
it is.

This refactor will also allow me to add in plugin functionality for
deploying Redis nodes via tools such as Nomad and Docker, and have them
available in the interface for inclusion in the suite.


# Requirements

As RS is written in Go you need Go installed. Once cloned, you will need to
install a few dependencies:

* go get "github.com/kelseyhightower/envconfig"
* go get "github.com/therealbill/airbrake-go"
* go get "github.com/therealbill/libredis/client"
* go get "github.com/therealbill/libredis/info"
* go get "github.com/zenazn/goji"

Then you can execute `go build` in the `redskull-controller` directory.

# Installation

Assuming you have Git and Go (sounds like a techie oriented convenience
store - "the Git and Go") installed, installing Red Skull is fairly
simple. The dependencies are listed in the Godeps file. 
```shell
go get github.com/therealbill/redskull/redskull-controller
```

And there should be a binary at `$GOPATH/bin/redskull-controller` with
the source in the usual location.

Now, assuming you have a sentinel config at /etc/redis/sentinel.conf, it
will be up and running on localhost port 8000.

There is also a Makefile now. Targets are "redskull" "dist-tar", and
"docker-image".

# Running Red Skull

Red Skull expects to find the sentinel config file in
/etc/redis/sentinel.conf.  You can, however, alter this by the setting
the environment variable REDSKULL_SENTINELCONFIGFILE.

RS currently expects the html directory to be in the same location as
the binary. For example you can do create a directory named
`/usr/redskull`, place the redskull binary in it, and copy the
html directory to it, then launch `./redskull` and it should work.
You'll find it running on port 8000, Alternatively you can configure the
location of the HTML directory via `REDSKULL_TEMPLATEDIRECTORY`,


# Calling the API

Err, for now look in main.go to see the URLs and whether you need to do
a GET, PUT, DEL, or POST for that call. Most of it is pretty simple.
I've just not documented it yet as I prefer to do it once things
stabilize. If you want to help get that jumpstarted pull requests are
welcome. :)


Can you use it for "production use"? Yes. Will it destroy your setup?
Not likely. It only executes read-only commands unless you click the
button to make a change.
