# pdwfs

[![Build Status](https://travis-ci.org/cea-hpc/pdwfs.png?branch=master)](https://travis-ci.org/cea-hpc/pdwfs)

## PaDaWAn project

PaDaWAn (for Parallel Data Workflow for Analysis) is a [CEA](http://www.cea.fr) project that aims at providing a lightweight and non-intrusive software infrastructure to facilitate *in transit* execution of file-based HPC simulation workflows. 

## pdwfs

One component of the project is pdwfs (pronounced "*padawan-fs*"): a preload library implementing a simplified file system in user space suitable for intercepting “bulk I/Os” typical of HPC simulations and leveraging Redis in-memory database for data staging.


## In situ / in transit HPC workflows
Within the HPC community, in situ data processing is getting quite some interests as a potential enabler for future exascale-era simulations. 

While the original in situ approach consists in executing data processing within the same address space as the simulation and sharing the resources with it, the loosely-coupled flavor of in situ, also called in transit, offers a great deal of flexibility to accommodate various types of workflows to be run “in situ”, that is on the supercomputer while the simulation is running.

## Work in progress...

pdwfs is a work in progress and still fairly experimental.

## License

pdwfs is distributed under the Apache License (Version 2.0).

