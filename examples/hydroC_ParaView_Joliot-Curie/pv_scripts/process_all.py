#!/usr/bin/env pvpython

# creates a worker pool and schedule ParaView tasks (paraview_run.py)

import os
import sys
import subprocess
import multiprocessing

## input:
if len(sys.argv) == 4:
    pv_script_path = str(sys.argv[1])
    nb_files = int(sys.argv[2])
    nb_workers = int(sys.argv[3])
else:
    print "usage: ./process_all.py pv_script_path nb_files"
    print "  with"
    print "    pv_script_path   - path to Paraview script"
    print "    nb_files         - number of VTK files to process"
    print "    nb_workers       - number of multiprocessing Pool workers"
    sys.exit(1)

FILE_INDICES = [str(i+1).zfill(2) for i in range(nb_files)]

try:
    os.mkdir("images")
except OSError:
    pass

def task(idx):
    filename = 'Dep/0000/{0}/Hydro_00{0}.pvtr'.format(idx)
    # pvpython: --no-mpi prevents a SLURM error related to pmi2/pmi1
    # the try...except is a fix for python bug #9400
    try:
        subprocess.check_call("pvpython --no-mpi" + " " + pv_script_path + " " + filename + " " + idx, shell=True)
    except subprocess.CalledProcessError as e:
        raise Exception(str(e))

workers = multiprocessing.Pool(processes=nb_workers)

futures = []

for idx in FILE_INDICES:
    futures.append(workers.apply_async(task, (idx,)))

for future in futures:
    future.wait()

