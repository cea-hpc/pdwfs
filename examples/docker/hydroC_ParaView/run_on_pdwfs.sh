#!/bin/bash

# Start two Redis instances in background to stage data (+deactivate Redis snapshotting)
redis-server --port 6379 --daemonize yes --save ""
redis-server --port 6380 --daemonize yes --save ""

# Export server addresses to pdwfs
export PDWFS_REDIS=localhost:6379,localhost:6380

# Wrap the simulation command with pdwfs intercepting IO in Dep/ directory
# Note: redirecting stderr to /dev/null as a read error appears at each iteration 
# I have no explanation yet, but since it does not affect the run, I am shutting it down for the example
pdwfs -p Dep -- mpirun hydro -i hydro_input.nml 2> /dev/null

# Wrap the post-processing command with pdwfs intercepting IO in Dep/ directory,
# this processing will save images on disk in the images/ folder which is not intercepted by pdwfs
pdwfs -p Dep -- ./process_all.py

# Build the movie from images on disk
ffmpeg -i images/test_%02d.jpg -vcodec libx264 -pix_fmt yuv420p -profile:v baseline -level 3 /output/hydro.mp4

# No data to clean up on disk, just shutdown Redis
#redis-cli -p 6379 shutdown
#redis-cli -p 6380 shutdown
