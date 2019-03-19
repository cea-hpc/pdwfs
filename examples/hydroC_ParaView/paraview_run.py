#!/usr/bin/env pvpython

import os
import sys
import time

from paraview.simple import *

## input:
if len(sys.argv) == 3:
    filename = str(sys.argv[1])
    idx = str(sys.argv[2])
else:
    print "usage: ./paraview_process_file.py pvtr-file index"
    print "  with"
    print "    pvtr-file  - partitioned legacy VTK file produced by Hydro"
    print "    index      - index appended to image filename"
    sys.exit(1)

while True:
    if os.path.exists(filename):
        break
    time.sleep(2.0)

print "Processing: ", filename

reader = OpenDataFile(filename)

view = CreateRenderView()
view.ViewSize = [1000,600]

display = Show(reader, view)

view.ResetCamera()

display.SetRepresentationType('Surface')
ColorBy(display, ('CELLS', 'varIP'))

display.SetScalarBarVisibility(view, True)

varIPLUT = GetColorTransferFunction('varIP')
varIPLUT.RGBPoints = [0.0, 0.231373, 0.298039, 0.752941, 0.125, 0.865003, 0.865003, 0.865003, 0.25, 0.705882, 0.0156863, 0.14902]
varIPLUT.ScalarRangeInitialized = 1.0

img_filename = 'images/test_{0}.jpg'.format(idx)

view.WriteImage(img_filename, "vtkJPEGWriter", 1)

print "Produced: ", img_filename


