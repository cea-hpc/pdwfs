+++
title = "IOR Benchmark"
description = ""
weight = 2
+++

This example illustrates the use of pdwfs with the HPC I/O benchmark tool [IOR](https://github.com/hpc/ior).
<!--more-->

The example runs in a Jupyter notebook and everything you need is packaged in a Dockerfile. 

We also provide a Makefile which does the plumbing for you. So, to build and run the container, just type:

```
$ git clone https://github.com/cea-hpc/pdwfs
$ cd pdwfs
$ make -C examples/ior_benchmark run
```

When the Jupyter notebook server is up and running, open your browser on your host at http://localhost:8888, open the notebook *ior_example.ipynb* and follow the steps.

This example also allows to run the benchmark in a non-interactive way and save the results in the output directory. To run the benchmark this way, just type:
```
$ make -C examples/ior_benchmark bench
```
Once finished it will show you the results in your browser. 