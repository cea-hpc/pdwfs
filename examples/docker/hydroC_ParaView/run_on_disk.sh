#!/bin/bash

# Execute the simulation
# Note: redirecting stderr to /dev/null as a read error appears at each iteration 
# I have no explanation yet, but since it does not affect the run, I am shutting it down for the example
mpirun hydro -i hydro_input.nml 2> /dev/null

# Post-process output data to produce movie images
./process_all.py

# Build the movie (options are for better compatibility)	
ffmpeg -i images/test_%02d.jpg -vcodec libx264 -pix_fmt yuv420p -profile:v baseline -level 3 /output/hydro.mp4

# Clean up output data when done
#rm -rf images/ Dep/ Hydro.pvd
