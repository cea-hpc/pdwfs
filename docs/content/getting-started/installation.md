+++
title = "Installation"
description = ""
weight = 3
+++

The following steps will help you install pdwfs and its dependencies.
<!--more-->

## Requirements

pdwfs only dependencies are:

* Go version ≥ 1.11 (for building only)
* Redis (recommended version ≥ 5.0.3)

### Installing Go

### Installing Redis


## Installation

### From binary distribution

### From sources
To build pdwfs from source (assuming Go is installed) :
```
$ git clone https://github.com/cea-hpc/pdwfs
$ cd pdwfs
$ make
```
Binary distributions are also available for Linux system and x86_64 architecture in the [releases](http://github.com/cea-hpc/pdwfs/releases) page.

To run the test suite, you will need a running Redis instance on the default host and port. Just type the following command to have an instance running in the background:
```
$ redis-server --daemonize yes
```
Then:
```
$ make test
```
To install pdwfs:
```
$ make PREFIX=/your/installation/path install
```
Default prefix is /usr/local.


We also provide a development Dockerfile based on an official Go image from DockerHub. To build and run the container:
```
$ make -C docker run
```
The working directory in the container is a mounted volume on the pdwfs repo on your host, so to build pdwfs, just use the Makefile as previously described.

NOTE: if you encounter permission denied issue when building pdwfs in the container that's probably because the non-root user and group IDs set in the Dockerfile do not match your UID and GID. Change the UID and GID values to yours in the Dockerfile and re-run the above command.

## Quick start

First, start a default Redis instance in the background.
```
$ redis-server --daemonize yes
``` 
Then, assuming your simulation will write its data into the output/ directory, simply wrap the execution command of your simulation with pdwfs command-line script like this:
```
$ pdwfs -p output/ -- your_simulation_command
```
That's it ! pdwfs will transparently intercept low-level I/O calls (open, write, read, ...) on any file/directory within the output/ directory and send data to Redis, no data will be written on disk.

To process the simulation data, just run your processing tool the same way:
```
$ pdwfs -p output/ -- your_processing_command
```
To see the data staged within Redis (keys only) and check the memory used (and to give you a hint at how sweet Redis is):
```
$ redis-cli keys *
...
$ redis-cli info memory
...
```

Finally, to stop Redis (and discard all data staged in memory !):
```
$ redis-cli shutdown
```

## Configuration

[Follow instructions here]({{%relref "configuration.md"%}})
