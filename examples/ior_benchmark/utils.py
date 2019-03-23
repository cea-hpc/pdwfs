#!/usr/bin/env python

import itertools as it
from collections import namedtuple
from jinja2 import Template
import matplotlib.pyplot as plt


def build_ior_script(api, read, numTasks, filePerProc, collective, segmentCount, transferSize):
    """
    Build IOR script from the jinja2 template ior_script.jinja2 and arguments
    """
    with open("ior_script.jinja2", "r") as f:
        template = Template(f.read())

    with open("ior_script", "w") as f:
        f.write(template.render(api=api,
                                read=read, 
                                numTasks=numTasks,
                                filePerProc=filePerProc,
                                collective=collective,
                                segmentCount=segmentCount,
                                transferSize=transferSize))


def parse_ior_results(filename):
    """
    Parse IOR results in file given by filename and return a Pandas dataframe
    """
    import re
    import pandas

    start_line = None
    end_line = None
    with open(filename,'r') as f: 
        for i, line in enumerate(f.readlines()):
            if re.search("Summary of all tests:", line):
                 start_line = i + 1
            if re.search("Finished", line):
                end_line = i - 1

    return pandas.read_csv(filename, sep='\s+', skiprows=start_line, nrows=end_line-start_line)


def plot_results(df_disk, df_pdwfs, title=None, filename=None):
    """
    Plot max write rate vs transfer size
    """
    plt.xlabel("Transfer Size (MiB)")
    plt.ylabel("Measured Write Rate (MiB/s)")
    prefix = "IOR Write Rate Test Results"
    title = prefix + " - " + title if title else prefix 
    plt.title(title)

    fig = plt.gcf()
    fig.set_size_inches(16, 6.5)

    plt.plot(df_disk["xsize"] / 1.e6, df_disk["Max(MiB)"],'-o',label="disk")
    plt.plot(df_pdwfs["xsize"] / 1.e6, df_pdwfs["Max(MiB)"],'-o',label="pdwfs")

    plt.legend(["disk","pdwfs"],loc="upper right")

    if filename:
        plt.savefig(filename)
        plt.clf()


def offline_test(api, version):
    read = "0"        # 1: perform read benchmark
    numTasks="2"      # number of parallel processes
    filePerProc="0"   # 1: write one file per processes
    collective="1"    # 1: enable collective IO operations (MPIIO, HDF5 only)
    segmentCount="1"  # see previous schematic
    transferSize = ["512k", "1m", "3m", "5m", "7m", "10m","25m","35m", "50m","60m","75m","85m", "100m","115m","125m","150m","175m","200m", "225m", "250m"]
    build_ior_script(api, read, numTasks, filePerProc, collective, segmentCount, transferSize)

    import subprocess
    subprocess.check_call(["bash", "run/bench.sh"])

    df_disk = parse_ior_results("run/ior_results_disk.out")
    df_pdwfs = parse_ior_results("run/ior_results_pdwfs.out")

    plot_results(df_disk, df_pdwfs, title=api + " with collective operations - " + verion, save=True)

if __name__ == '__main__':
    import sys
    offline_test(api=sys.argv[1], version=sys.argv[2])