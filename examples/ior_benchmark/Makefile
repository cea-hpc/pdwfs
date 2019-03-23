
# uncomment to mount pdwfs sources folder into the container (for debug/development)
# a modification is needed in the Dockerfile as well
DOCKER_RUN_OPT = -v $(shell pwd)/../../:/pdwfs

# uncomment to allow stracing in docker (for debug)
#DOCKER_RUN_OPT = --security-opt seccomp:unconfined $(DOCKER_RUN_OPT)

DOCKER_RUN = docker run $(DOCKER_RUN_OPT) -it --rm -p 8888:8888 -v $(shell pwd)/output:/output -v $(shell pwd)/notebook:/notebook --name ior-run ior

build:
	docker build -t pdwfs-base ../base_dockerfile
	docker build -t ior .

run: build
	$(DOCKER_RUN) jupyter-notebook
	

define BROWSER_PYSCRIPT
import os, webbrowser, sys
try:
	from urllib import pathname2url
except:
	from urllib.request import pathname2url
webbrowser.open("file://" + pathname2url(os.path.abspath(sys.argv[1])))
endef
export BROWSER_PYSCRIPT

BROWSER := python -c "$$BROWSER_PYSCRIPT"

bench: build
	$(DOCKER_RUN) ./bench.py $(shell git describe)	
	$(BROWSER) output/bench.html


connect:
	docker exec -it ior-run bash

