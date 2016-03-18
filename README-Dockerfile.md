# Using the Dockerfiles


The "standard" Dockerfile builds an image running Sentinel and Redskull under
Supervisord. If your Docker *host* already has Consul running, use this one. If
you need Consul to run in the Redskull container use `Dockerfile-consul`.

## Makefile for Ease of Use

The makefile has targets set up for with or without Consul.

# Assumptions

It assumes your `docker0` interface is left with the stock setup. If
you've modified it you will need to modify the JOIN IP in the
Dockerfile.

It also assumes an entirely stock Docker network config. On Port 8000 of
the container IP will be Red Skull, and Sentinel will be on the stock
port of 26379.

As RedSkull uses environment variables for config you can pass them in
the `docker run` command to change them if needed. Normally you won't
need to outside of the Consul Address.

