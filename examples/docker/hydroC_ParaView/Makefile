
# uncomment to mount pdwfs sources folder into the container (for debug/development)
# a modification is needed in the Dockerfile as well
DOCKER_RUN_OPT = -v $(shell pwd)/../../../:/pdwfs

# uncomment to allow stracing in docker (for debug)
#DOCKER_RUN_OPT = --security-opt seccomp:unconfined $(DOCKER_RUN_OPT)

build:
	docker build -t pdwfs-base ../base_dockerfile
	docker build -t hydro .

run: build
	docker run $(DOCKER_RUN_OPT) -it --rm -v $(shell pwd)/output:/output --name hydro-run hydro

connect:
	docker exec -it hydro-run bash

watch:
	@echo "<video preload controls autoplay loop src=hydro.mp4></video>" > output/video.html && python -m webbrowser -t "output/video.html"

