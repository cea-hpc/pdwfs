# pdwfs

[![Build Status](https://travis-ci.org/cea-hpc/pdwfs.png?branch=master)](https://travis-ci.org/cea-hpc/pdwfs)

pdwfs (pronounced "*padawan-f-s*", see below) is a preload library implementing a minimalist filesystem in user space suitable for intercepting *bulk* I/Os typical of HPC simulations and storing data in memory in a [Redis](https://redis.io) database.

pdwfs objective is to provide a very lightweight infrastructure to execute HPC simulation workflows without writing/reading any intermediate data to/from a (parallel) filesystem. This type of approach is known as *in transit* or *loosely-coupled in situ*, see the two next sections for further details.

pdwfs is written in [Go](https://golang.org) and C and runs on Linux systems only (we provide a Dockerfile for testing and development on other systems).

Though it's a work in progress and still at an early stage of development, it can already be tested with Parallel HDF5, MPI-IO and a POSIX-based  ParaView workflow. See Examples section below.


## PaDaWAn project

pdwfs is a component of the PaDaWAn project (for Parallel Data Workflow for Analysis), a [CEA](http://www.cea.fr) project that aims at providing building blocks of a lightweight and *least*-intrusive software infrastructure to facilitate *in transit* execution of HPC simulation workflows. 

The foundational work for this project was an initial version of pdwfs entierly written in Python and presented in the paper below:

- *PaDaWAn: a Python Infrastructure for Loosely-Coupled In Situ Workflows*, J. Capul, S. Morais, J-B. Lekien, ISAV@SC (2018).

## In situ / in transit HPC workflows
Within the HPC community, in situ data processing is getting quite some interests as a potential enabler for future exascale-era simulations. 

The original in situ approach, also called tightly-coupled in situ, consists in executing data processing routines within the same address space as the simulation and sharing the resources with it. It requires the simulation to use a dedicated API and to link against a library embedding a processing runtime. Notable in situ frameworks are ParaView [Catalyst](https://www.paraview.org/in-situ/), VisIt [LibSim](https://wci.llnl.gov/simulation/computer-codes/visit). [SENSEI](http://sensei-insitu.org) provides a common API that can map to various in situ processing backends.

The loosely-coupled flavor of in situ, or in transit, relies on separate resources from the simulation to stage and/or process data. It requires a dedicated distributed infrastructure to extract data from the simulation and send it to a staging area or directly to consumers. Compared to the tightly-coupled in situ approach, it offers greater  flexibility to adjust the resources needed by each application in the workflow (not bound to efficiently use the same resources as the simulation). It can also accommodate a larger variety of workflows, in particular those requiring memory space for data windowing (e.g. statistics, time integration).

This latter approach, loosely-coupled in situ or in transit, is at the core of pdwfs. 

## Dependencies

pdwfs only dependencies are:
- Go version ≥ 1.11 to build pdwfs
- Redis version ≥ 5.0.3


## Installation

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

## How does it work ?

pdwfs used the often-called "LD_PRELOAD trick" to intercept a set of I/O-related function calls provided by the C standard library (libc).

The pdwfs command-line script execute the user command passed in argument with the LD_PRELOAD environment variable set to the installation path of the pdwfs.so shared library.

Currently about 90 libc I/O calls are intercepted but only 50 of them are currently implemented by pdwfs. If your program uses an I/O call intercepted but not implemented, it will raise an error. In this case, please file an [issue](https://github.com/cea-hpc/pdwfs/issues) (or even better, send a pull request!).

The category of calls currently implemented are:
- basic POSIX and C standard I/O calls (open, close, read, write, lseek, access, unlink, stat, fopen, fread, fprintf, ...)
- additional I/O calls used by MPI-IO and Parallel HDF5 libraries (writev, pwrite, pwritev, readv, pread, preadv, statfs, ...)

## Performance

To address the challenge of being competitive with parallel filesystems, an initial set of design choices and trade-offs have been made:
- selecting the widely used database Redis to benefit from its mix of performance, simplicity and flexibility (and performance is an important part of the mix),
- files are sharded (or stripped) accross multiple Redis instances with a predefined layout (by configuration),
- file shards are sent/fetched in parallel using Go concurrency mechanism (goroutines),
- no central metadata server, metadata are distributed accross Redis instances,
- (planned) drastically limit the amount of metadata and metadata manipulations by opting not to implement typical filesystem features such as linking, renaming and timestamping,
- (planned) implement write buffers and leverages Redis pipelining feature,

With this set of choices, we expect our infrastructure to be horizontally scalable (adding more Redis instances to accomodate higher loads) and to accomodate certain I/O loads that are known to be detrimental for parallel filesystem (many files).

On the other hand, a few factors are known for impacting performance versus parallel filesystems:
- Redis uses TCP communications while parallel filesystems rely on RDMA,
- intercepting I/O calls and the use of CGO (system to call Go from C) adds some overhead to the calls,

Obvisouly, proper benchmarking at scale will have to be performed to assess pdwfs performances. Yet, considering these design choices and our past experience with PaDaWAn in designing in transit infrastructure, we are hopefull that we will get decent performances.

It is also noted that a significant difference with parallel filesystems is that pdwfs is a simple infrastructure in user space that can be easily adjusted on a per-simulation or per-workflow basis for efficiency.

## Validation

Test cases have been successfully run so far with the followig codes and tools:
- [IOR](https://github.com/hpc/ior) benchmark with POSIX, parallel HDF5 and MPI-IO methods  (OpenMPI v2),
- [HydroC](https://github.com/HydroBench/Hydro) a 2D structured hydrodynamic mini-app using POSIX calls to produce VTK files,
- [ParaView](https://www.paraview.org/in-situ/) VTK file reader.


## Examples
We provide a set of Dockerfiles to test on a laptop the codes and tools described in the Validation section.

- **Example 1**: HydroC + ParaView + FFmpeg workflow

Check the README in the example/HydroC_ParaView directory or just go ahead and type:
```
$ make -C examples/HydroC_ParaView run
```
You can go grab some coffee, building the container takes a while... 

- **Example 2**: IOR benchmark

Again, check the README in the corresponding directory or go ahead and type:
```
$ make -C examples/IOR_benchmark run
```
Yep, you can go grab a second coffee...

## Known limitations

- Works only for dynamically linked executables,
- Most core or shell utilities for file manipulations (e.g. ls, rm, redirections) requires particular libc calls not implemented,

## License

pdwfs is distributed under the Apache License (Version 2.0).

## Acknowledgements

This work was conducted in collaboration with [CINES](http://www.cines.fr).

