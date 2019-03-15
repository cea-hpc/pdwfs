#!/usr/bin/env python

import os
import subprocess
import multiprocessing

NB_FILES = 98
FILE_INDICES = [str(i+1).zfill(2) for i in range(NB_FILES)]

try:
    os.mkdir("images")
except OSError:
    pass

def task(idx):
    filename = 'Dep/0000/{0}/Hydro_00{0}.pvtr'.format(idx)
    subprocess.check_call("./paraview_run.py " + filename + " " + idx, shell=True)


workers = multiprocessing.Pool()

futures = []

for idx in FILE_INDICES:
    futures.append(workers.apply_async(task, (idx,)))

for future in futures:
    future.wait()

