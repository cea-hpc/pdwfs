#!/bin/bash
#MSUB -q skylake
#MSUB -r pdwfs
#MSUB -N 3
#MSUB -n 144
#MSUB -x
#MSUB -T 600
#MSUB -m scratch

module load flavor/paraview/osmesa paraview
module load ffmpeg/4.0.2

# for debugging
#set -x

# to use pdwfs version installed on Joliot-Curie
#module load pdwfs

# to use the current development version of pdwfs (this repo)
module load redis
export PATH="$(realpath ../../../build/bin):$PATH"

# directory containing ParaView python script
SRC_DIR=$(realpath ../pv_scripts)

# set PATH to hydro executable
export PATH="$(realpath ../bin):$PATH"

# set up job scratch directory
JOB_SCRATCH=$CCCSCRATCHDIR/hydroC_ParaView_with_pdwfs
rm -rf $JOB_SCRATCH
mkdir -p $JOB_SCRATCH

# copy input deck in job scratch directory
cp hydro_input.nml $JOB_SCRATCH

cd $JOB_SCRATCH

# Initialize pdwfs data staging (launch 16 Redis instances on one node, bind them to high-speed network interface)
pdwfs-slurm init -N 1 -n 16 -i ib0.8fff

# previous call generates a session file with environement variables to be sourced for pdwfs
source ./pdwfs.session

# Launch Hydro simulation and return immediately (run in background)
# the run is set up to produce 98 timesteps with output data in separate VTK files in a Dep/ directory
# the hydro executable has been prepended with pdwfs command to intercept I/O in Dep/ directory (directory used by Hydro)
ccc_mprun -N 1 -n 48 pdwfs -p Dep/ hydro -i hydro_input.nml &

# start the post-processing in parallel with the simulation
# process_all.py: starts a pool of 48 workers based on python multiprocessing module and schedules 98 tasks (one per simulation timestep)
# paraview_run.py: ParaView script to process one simulation timestep and produce an image (it will wait for the timestep file to be available)
# the process_all.py script has been prepended with pdwfs command to intercept I/O in Dep/ directory
ccc_mprun -N 1 -n 1 -c 48 pdwfs -p Dep/ pvpython $SRC_DIR/process_all.py $SRC_DIR/paraview_run.py 98 48


# Gracefully shuts down pdwfs Redis instances (it will discard all intermediate data written in Dep/ directory)
pdwfs-slurm finalize

# wait for all background task to complete
wait 

# Finally build the movie (options are for better compatibility)
# all images produces by the ParaView script are in the images/ directory	
ffmpeg -i images/test_%02d.jpg -vcodec libx264 -pix_fmt yuv420p -profile:v baseline -level 3 hydro_movie.mp4

