+++
title = "pdwfs"
description = ""
+++

# ![pdwfs](/logo.png?height=120px)

##### TL;DR A Redis-backed distributed in-memory filesystem in user space to accelerate HPC workflows
pdwfs (pronounced "*padawan-f-s*") is a preload library emulating a filesystem in user space adapted for intercepting *bulk* I/Os typical of HPC simulations. It uses a single or multiple Redis instances as a distributed infrastructure for staging data in memory.

pdwfs objective is to provide a very lightweight infrastructure to execute HPC simulation workflows without writing/reading any intermediate data to/from a (parallel) filesystem. This type of approach is known as in transit or loosely-coupled in situ, see the two next sections for further details.

{{% notice info %}}
pdwfs is a work in progress and still at an early stage of development, yet it can already be used with Parallel HDF5 and MPI-IO, [go to Examples section]({{%relref "examples/_index.md"%}}).
{{% /notice %}}

pdwfs is written in [Go](https://golang.org) and C and runs on Linux systems only.

## Overview

![overview](/pdwfs-overview.png?height=350px)



## Main features

* features...

## PaDaWAn project

pdwfs is a component of the PaDaWAn project (for **Pa**rallel **Da**ta **W**orkflow for **An**alysis), a [CEA](http://www.cea.fr) project that aims at providing building blocks of a lightweight and *least*-intrusive software infrastructure to facilitate in transit execution of HPC simulation workflows. 

The foundational work for this project was an initial version of pdwfs entierly written in Python and presented in the paper:  
[*PaDaWAn: a Python Infrastructure for Loosely-Coupled In Situ Workflows*, J. Capul, S. Morais, J-B. Lekien, ISAV@SC (2018)](https://www.google.com/url?sa=t&rct=j&q=&esrc=s&source=web&cd=1&cad=rja&uact=8&ved=2ahUKEwjJn9npiprhAhWKqZ4KHbviDBcQFjAAegQIARAC&url=https%3A%2F%2Fsc18.supercomputing.org%2Fproceedings%2Fworkshops%2Fworkshop_files%2Fws_isav116s3-file1.pdf&usg=AOvVaw3KqucnBMfYd__FLgMGzM2e)


## Post processing vs. in situ / in transit processing
![schema-processing](/schema-processing.png?height=400px)
Within the HPC community, in situ processing is getting significant interest as a potential enabler for future exascale-era simulations. 

The original in situ approach, also called tightly-coupled in situ, consists in executing data processing routines within the same address space as the simulation and sharing the resources with it. It requires the simulation to use a dedicated API and to link against a library embedding a processing runtime. Notable in situ frameworks are ParaView [Catalyst](https://www.paraview.org/in-situ/), VisIt [LibSim](https://wci.llnl.gov/simulation/computer-codes/visit) and [Ascent](https://github.com/Alpine-DAV/ascent).

The loosely-coupled flavor of in situ, or in transit, relies on separate resources from the simulation to stage and/or process data. It requires a dedicated distributed infrastructure to extract data from the simulation and send it to a staging area or directly to consumers. Compared to the tightly-coupled in situ approach, it offers greater  flexibility to adjust the resources needed by each application in the workflow (not bound to efficiently use the same resources as the simulation). It can also accommodate a larger variety of workflows, in particular those requiring memory space for data windowing (e.g. statistics, time integration).

This latter approach, loosely-coupled in situ or in transit, is at the core of pdwfs. 



