#!/bin/bash

module load intel mpi

# Run the job without pdwfs (using the scratch filesystem to store files)
#ccc_mprun -K -p skylake -N 2 -x -T 600 -m scratch ./job_without_pdwfs.sh

# Run with pdwfs (needs an additional node to stage the files)
ccc_mprun -K -p skylake -N 3 -x -T 600 -m scratch ./job_with_pdwfs.sh
