#!/bin/bash
# SBATCH --job-name=pdwfs
# SBATCH --nodes=12
# SBATCH --exclusive

# This script demonstrates how to run the I/O benchmarking tool ior 
# with pdwfs using SLURM. 
# pdwfs comes with the pdwfs-slurm CLI tool to manage Redis instances

# pre-requisites: 
# - pdwfs and Redis are installed and available in PATH
# - ior is installed with MPI support

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