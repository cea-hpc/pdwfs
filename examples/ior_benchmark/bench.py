#!/usr/bin/env python

import os
import sys
import glob
import subprocess
from datetime import datetime

# override the matplotlib import in utils.py
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot

# load local utils.py module
import utils

bench_script = """
#!/bin/bash

cd run

echo "    Executing the benchmark on disk"
mpirun ior -f ../ior_script > /output/ior_disk.out

echo "    Executing the benchmark with pdwfs"
# start a new Redis instance in the background
redis-cli shutdown 2> /dev/null
redis-server --daemonize yes --save "" > /dev/null

# wrap the previous command with pdwfs -p . (indicating the current directory as the intercepted directory)
pdwfs -p . -- mpirun ior -f ../ior_script > /output/ior_pdwfs.out
"""

def run_bench(api, version):

    bench_title = api + " IOR benchmark - pdwfs " + version + " - " + str(datetime.utcnow()) + " UTC"
    print "Running:", bench_title

    read = "0"        # 1: perform read benchmark
    numTasks="2"      # number of parallel processes
    filePerProc="0"   # 1: write one file per processes
    collective="1"    # 1: enable collective IO operations (MPIIO, HDF5 only)
    segmentCount="1"  # see previous schematic
    transferSize = ["512k", "1m", "3m", "5m", "7m", "10m","25m","35m", "50m","60m","75m","85m", "100m","115m","125m","150m","175m","200m", "225m", "250m"]
    utils.build_ior_script(api, read, numTasks, filePerProc, collective, segmentCount, transferSize)

    with open("run/bench.sh", "w") as f:
        f.write(bench_script)

    subprocess.check_call(["bash", "run/bench.sh"])

    print "    Parsing and saving the results in a plot"
    df_disk = utils.parse_ior_results("/output/ior_disk.out")
    df_pdwfs = utils.parse_ior_results("/output/ior_pdwfs.out")

    os.rename("/output/ior_disk.out", "/output/ior_" + api + "_disk.out")
    os.rename("/output/ior_pdwfs.out", "/output/ior_" + api + "_pdwfs-" + version + ".out")

    matplotlib.use('Agg')
    filename = "ior_" + api + "_pdwfs-" + version + ".png"
    utils.plot_results(df_disk, df_pdwfs, title=bench_title, filename="/output/" + filename)

    with open("/output/bench.html", "a") as f:
        f.write("<img src=" + filename + ">\n")

if __name__ == '__main__':

    for f in glob.glob("/output/*"):
        os.remove(f)

    version = sys.argv[1]
    run_bench("POSIX", version)
    run_bench("MPIIO", version)
    run_bench("HDF5", version)
