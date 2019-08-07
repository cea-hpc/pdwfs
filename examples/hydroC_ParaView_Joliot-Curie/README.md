# HydroC/ParaView/FFmpeg workflow on TGCC's Joliot-Curie

This repository provides an example workflow that uses pdwfs to store intermediate files in memory and which runs on TGCC's Joliot-Curie supercomputer.

The workflow applications are:
- HydroC: the C version of the open source 2D hydrodynamic benchmarking application [Hydro](https://github.com/HydroBench/Hydro),
- ParaView: visualization and graphics processing framework,
- FFmpeg: video processing,

The workflow is a simple pipeline which produces a small MP4 video of the simulation:

HydroC => ParaView (script) => FFmpeg 

pdwfs is used to store intermediate files of the first part of the workflow, ie files produced by Hydro and read by ParaView.

To run the workflow:

On your local machine with internet access and credentials to Joliot-Curie, run the following: 

```bash
$ make JOLIOTCURIE_LOGIN=your_login prep
``` 

It will fetch the Hydro source code from Github and copy the example directory on Joliot-Curie (it will prompt for your password).

Log in to Joliot-Curie, cd into the example directory and build Hydro (mpicc should be available in your PATH, which is the default behaviour): 
```bash
$ ssh your_login@irene-fr.ccc.cea.fr
$ cd examples/pdwfs/examples/hydroC_ParaView_Joliot-Curie
$ make build
```

Then follow the instructions in ```job/README.md``` to run the job without or with pdwfs.

