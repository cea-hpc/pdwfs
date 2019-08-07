# Workflow Example: HydroC + ParaView + FFmpeg

This example illustrates the use of pdwfs to produce movie images  from an [HydroC](https://github.com/HydroBench/Hydro) simulation without writing any simulation data on disk.

The example runs on a laptop and everything you need is packaged in a Dockerfile.

We also provide a Makefile which does the plumbing for you. So, to build and run the container, just type:
```
$ git clone https://github.com/cea-hpc/pdwfs
$ cd pdwfs
$ make -C examples/docker/hydroC_ParaView run
```
Once you are in the container, just follow the help message to run the workflow with pdwfs.

Once the workflow has run, go back to your host (not the container) and type the following command to watch the movie you just produced:

```
$ make -C examples/docker/hydroC_ParaView watch
```