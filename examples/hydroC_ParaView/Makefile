
# uncomment to mount pdwfs sources folder into the container (for debug/development)
# a modification is needed in the Dockerfile as well
DOCKER_RUN_OPT = -v $(shell pwd)/../../:/pdwfs

# uncomment to allow stracing in docker (for debug)
#DOCKER_RUN_OPT = --security-opt seccomp:unconfined $(DOCKER_RUN_OPT)

build:
	docker build -t pdwfs-base ../base_dockerfile
	docker build -t hydro .

run: build
	docker run $(DOCKER_RUN_OPT) -it --rm -v $(shell pwd)/output:/output --name hydro-run hydro

connect:
	docker exec -it hydro-run bash


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

watch:
	@echo "<video preload controls autoplay loop src=hydro.mp4></video>" > output/video.html && $(BROWSER) output/video.html

