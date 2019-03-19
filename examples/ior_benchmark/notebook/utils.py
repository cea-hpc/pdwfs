import itertools as it
from collections import namedtuple
from jinja2 import Template

IORCase = namedtuple("IORCase", 
                     field_names=["numTasks", "filePerProc", "collective", "segmentCount", "transferSize"])

def make_ior_script(api, numTasks, filePerProc, collective, segmentCount, transferSize):
    matrix = list(it.product(numTasks, filePerProc, collective, segmentCount, transferSize))
    cases = [IORCase(*case) for case in matrix]

    with open("ior_script.jinja2", "r") as f:
        template = Template(f.read())

    with open("ior_script_" + api, "w") as f:
        f.write(template.render(api=api, cases=cases))