# Workflow Example: HydroC + ParaView + FFmpeg

This example illustrates the use of pdwfs to produce movie images  from a simulation without writing any simulation output data on disk.

It can run on a laptop and only requires Docker:

```
$ git clone https://github.com/cea-hpc/pdwfs
$ cd pdwfs
$ make -C examples/hydroC_ParaView run
```
The last command will build the container the first time it is executed. As it is compiling OpenMPI and downloading Go and ParaView, you'll have time to grab some coffee...

Once the build is complete, it runs the container. Just follow the help message to run the workflow with pdwfs.

Once the workflow has run, go back to your host (not the container) and type the following command to watch the movie you just produced:

```
$ make -C examples/hydroC_ParaView watch
```