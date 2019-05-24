# pdwfs

[![Build Status](https://travis-ci.org/cea-hpc/pdwfs.png?branch=master)](https://travis-ci.org/cea-hpc/pdwfs)

pdwfs (we like to pronounce it "*padawan-f-s*", see [below](#padawan-project)) is a preload library implementing a distributed in-memory filesystem in user space suitable for intercepting *bulk* I/O workloads typical of HPC simulations. It is using [Redis](https://redis.io) as the backend memory store.

pdwfs objective is to provide a lightweight infrastructure to execute HPC simulation workflows without writing/reading any intermediate data to/from a (parallel) filesystem, but rather staging it in memory. This type of approach is known as *in transit* or *loosely-coupled in situ* and is further explained in  a [section](#in-situ-and-in-transit-hpc-workflows) below.

pdwfs is written in [Go](https://golang.org) and C and runs on Linux systems only (we provide a Dockerfile for testing and development on other systems).


## Dependencies

pdwfs only dependencies are:

Build:
- Go version ≥ 1.11

Runtime (and testing):
- Redis (version ≥ 5.0.3 is recommended)


## Installation

### From binary distribution

A binary distribution is available for Linux system and x86_64 architecture in the [releases](http://github.com/cea-hpc/pdwfs/releases) page.

The following steps will install pdwfs and make it available in your PATH.

```bash
$ wget -O pdwfs.tar.gz https://github.com/cea-hpc/pdwfs/releases/download/v0.2.0/pdwfs-v0.2.0-linux-amd64.tar.gz
$ mkdir /usr/local/pdwfs
$ tar xf pdwfs.tar.gz --strip-component=1 -C /usr/local/pdwfs
$ export PATH="/usr/local/pdwfs/bin:$PATH" 
```

### From source
A Go distribution (version ≥ 1.11) is required to build pdwfs from source (instructions can be found on the [Go download page](https://golang.org/dl/)).

To build pdwfs develop branch (default branch):
```bash
$ git clone https://github.com/cea-hpc/pdwfs
$ cd pdwfs
$ make PREFIX=/usr/local/pdwfs install
```

### Using Spack

pdwfs and its dependencies (Redis and Go) can be installed with the package manager [Spack](https://spack.io).

NOTE: at the time of this writing, the latest Spack release (v0.12.1) does not have the Redis package, it is only available in the develop branch. Still, the Redis package python file can easily be copy-pasted in your Spack installation.

A Spack package for pdwfs is not yet available in Spack upstream repository, but is available [here](https://github.com/cea-hpc/pdwfs/releases/download/v0.2.0/pdwfs-spack.py).

To add pdwfs package to your Spack installation, you can proceed as follows:

```bash
$ export DIR=$SPACK_ROOT/var/spack/repos/builtin/pdwfs
$ mkdir $DIR
$ wget -O $DIR/package.py https://github.com/cea-hpc/pdwfs/releases/download/v0.2.0/pdwfs-spack.py
$ spack spec pdwfs  # check everything is OK
```

## Testing

To run the test suite, you will need a Redis installation and make sure ```redis-server``` and ```redis-cli``` binaries are available in your PATH. Then:

```bash
$ make test
```

To allow development and testing on a different OS (e.g. MacOS), we provide a development Dockerfile based on an official CentOS image from DockerHub. To build and run the container:

```bash
$ make -C docker run
```

The working directory in the container is a mounted volume on the pdwfs repo on your host, so to build pdwfs inside the container, just use the Makefile as previously described (ie. ```make test```).

NOTE: if you encounter permission denied issue when building pdwfs in the container that's probably because the non-root user and group IDs set in the Dockerfile do not match your UID and GID. Change the UID and GID values to yours in the Dockerfile and re-run the above command.

## Quick start

First, start a default Redis instance in the background.

```bash
$ redis-server --daemonize yes
``` 

Then, assuming your simulation will write its data into the ```output``` directory, simply wrap the execution command of your simulation with pdwfs command-line script like this:

```bash
$ pdwfs -p output/ -- your_simulation_command
```
That's it ! pdwfs will transparently intercept low-level I/O calls (open, write, read, ...) on any file/directory within the output/ directory and send data to Redis, no data will be written on disk.

To process the simulation data, just run your processing tool the same way:

```bash
$ pdwfs -p output/ -- your_processing_command
```

To see the data staged within Redis (keys only) and check the memory used (and to give you a hint at how nice Redis is):

```bash
$ redis-cli keys *
...
$ redis-cli info memory
...
```

Finally, to stop Redis (and discard all data staged in memory !):

```bash
$ redis-cli shutdown
```

## Running pdwfs in a SLURM job

pdwfs comes with a specialized CLI tool called ```pdwfs-slurm``` that simplifies the deployment of Redis instances in a SLURM job.

```pdwfs-slurm``` has two subcommands: ```init``` and ```finalize```. 

The following script illustrates the use of pdwfs and ```pdwfs-slurm``` to run the I/O benchmarking tool [ior](https://github.com/hpc/ior) (the script is available in examples/ior_slurm). The script assumes that pdwfs, Redis and ior are installed and available in the PATH.

ior_pdwfs.sh:
```bash
#!/bin/bash
# SBATCH --job-name=pdwfs
# SBATCH --nodes=12
# SBATCH --exclusive

# Initialize the Redis instances:
# - 32 instances distributed on 4 nodes (8 per node)
# - bind Redis servers to ib0 network interface
pdwfs-slurm init -N 4 -n 8 -i ib0

# pdwfs-slurm produces a session file with some environment variables to source
source pdwfs.session

# ior command will use MPI-IO in collective mode with data blocks of 100 MBytes
IOR_CMD="ior -a MPIIO -c -t 100m -b 100m -o $SCRATCHDIR/testFile"

# pdwfs command will forward all I/O in $SCRATCHDIR in Redis instances
WITH_PDWFS="pdwfs -p $SCRATCHDIR"

# Execute ior benchmark on 128 tasks
srun -N 8 --ntasks-per-node 16 $WITH_PDWFS $IOR_CMD

# gracefully shuts down Redis instances
pdwfs-slurm finalize

# pdwfs-slurm uses srun in background to execute Redis instances
# wait for background srun to complete
wait
```

This script can be run with SLURM ```sbatch``` or with ```salloc``` as follows:
```bash
$ salloc -N 12 --exclusive ./ior_pdwfs.sh
```

## How does it work ?

pdwfs used the often-called "LD_PRELOAD trick" to intercept a set of I/O-related function calls provided by the C standard library (libc).

The pdwfs CLI script execute the user command passed in argument with the LD_PRELOAD environment variable set to the installation path of the pdwfs.so shared library.

Currently about 90 libc I/O calls are intercepted but only 50 of them are currently implemented by pdwfs. If your program uses an I/O call intercepted but not implemented, it will raise an error. In this case, please file an [issue](https://github.com/cea-hpc/pdwfs/issues) (or send a pull request!).

The category of calls currently implemented are:
- basic POSIX and C standard I/O calls (open, close, read, write, lseek, access, unlink, stat, fopen, fread, fprintf, ...)
- additional I/O calls used by MPI-IO and Parallel HDF5 libraries (writev, pwrite, pwritev, readv, pread, preadv, statfs, ...)

## Performance

To address the challenge of being competitive with parallel filesystems, an initial set of design choices and trade-offs have been made:
- no central metadata server, metadata are distributed accross Redis instances,
- drasticaly limiting the amount of metadata and metadata manipulations by opting not to implement typical filesystem features such as linking, renaming and timestamping
- selecting the widely used database Redis to benefit from its mix of performance, simplicity and flexibility (and performance is an important part of the mix),
- files are stripped accross multiple Redis instances with a predefined layout (by configuration), no metadata query for read/write,
- being a simple infrastructure in user space allows adjustement and configuration on a per-simulation or per-workflow basis for increased efficiency.

With this set of choices, we expect our infrastructure to be horizontally scalable (adding more Redis instances to accomodate higher loads) and to accomodate certain I/O loads that are known to be detrimental for parallel filesystem (many files).

On the other hand, a few factors are known for impacting performance versus parallel filesystems:
- Redis uses TCP communications while parallel filesystems rely on RDMA,
- intercepting I/O calls and the use of CGO (system to call Go from C) adds some overhead to the calls,

Obvisouly, proper benchmarking at scale will have to be performed to assess pdwfs performances. Yet, considering these design choices and our past experience with PaDaWAn in designing in transit infrastructure, we are hopefull that we will get decent performances.


## Validation

Test cases have been successfully run so far with the followig codes and tools:
- [IOR](https://github.com/hpc/ior) benchmark with POSIX, parallel HDF5 and MPI-IO methods  (OpenMPI v2),
- [HydroC](https://github.com/HydroBench/Hydro) a 2D structured hydrodynamic mini-app using POSIX calls to produce VTK files,
- [ParaView](https://www.paraview.org/in-situ/) VTK file reader.


## Docker-packaged examples
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


## PaDaWAn project

pdwfs is a component of the PaDaWAn project (for Parallel Data Workflow for Analysis), a [CEA](http://www.cea.fr) project that aims at providing building blocks of a lightweight and *least*-intrusive software infrastructure to facilitate *in transit* execution of HPC simulation workflows. 

The foundational work for this project was an initial version of pdwfs entierly written in Python and presented in the paper below:

- *PaDaWAn: a Python Infrastructure for Loosely-Coupled In Situ Workflows*, J. Capul, S. Morais, J-B. Lekien, ISAV@SC (2018).

## In situ and in transit HPC workflows
Within the HPC community, in situ data processing is getting quite some interests as a potential enabler for future exascale-era simulations. 

The original in situ approach, also called tightly-coupled in situ, consists in executing data processing routines within the same address space as the simulation and sharing the resources with it. It requires the simulation to use a dedicated API and to link against a library embedding a processing runtime. Notable in situ frameworks are ParaView [Catalyst](https://www.paraview.org/in-situ/), VisIt [LibSim](https://wci.llnl.gov/simulation/computer-codes/visit). [SENSEI](http://sensei-insitu.org) provides a common API that can map to various in situ processing backends.

The loosely-coupled flavor of in situ, or in transit, relies on separate resources from the simulation to stage and/or process data. It requires a dedicated distributed infrastructure to extract data from the simulation and send it to a staging area or directly to consumers. Compared to the tightly-coupled in situ approach, it offers greater  flexibility to adjust the resources needed by each application in the workflow (not bound to efficiently use the same resources as the simulation). It can also accommodate a larger variety of workflows, in particular those requiring memory space for data windowing (e.g. statistics, time integration).

This latter approach, loosely-coupled in situ or in transit, is at the core of pdwfs. 


## License

pdwfs is distributed under the Apache License (Version 2.0).

## Acknowledgements

This work was conducted in collaboration with [CINES](http://www.cines.fr).

