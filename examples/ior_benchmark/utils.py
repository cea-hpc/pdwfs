#!/usr/bin/env python

import re
import itertools as it
from collections import namedtuple

from jinja2 import Template
import matplotlib.pyplot as plt
import pandas


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
    

    start_line = None
    end_line = None
    with open(filename,'r') as f: 
        for i, line in enumerate(f.readlines()):
            if re.search("Summary of all tests:", line):
                 start_line = i + 1
            if re.search("Finished", line):
                end_line = i - 1

    return pandas.read_csv(filename, sep='\s+', skiprows=start_line, nrows=end_line-start_line)


def plot_results(readOrWrite, df_disk, df_pdwfs, title=None, filename=None):
    """
    Plot max write rate vs transfer size
    """
    plt.style.use("ggplot")

    plt.xlabel("Transfer Size (MiB)")
    plt.ylabel("Measured " + readOrWrite + " Rate (MiB/s)")
    prefix = "IOR " + readOrWrite + " Rate Test Results"
    title = prefix + " - " + title if title else prefix 
    plt.title(title)

    fig = plt.gcf()
    fig.set_size_inches(16, 6.5)

    plt.plot(df_pdwfs["xsize"] / 1.e6, df_pdwfs["Max(MiB)"],'-o',label="pdwfs")
    if readOrWrite == "write":
        # plot only "write" results on disk, "read" results on disk are "polluted" by caching in RAM 
        plt.plot(df_disk["xsize"] / 1.e6, df_disk["Max(MiB)"],'-o',label="disk")
        plt.legend(["pdwfs", "disk"],loc="upper right")  
    else:
        plt.legend(["pdwfs"],loc="upper right")

    if filename:
        plt.savefig(filename)
        plt.clf()
