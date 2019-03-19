
# uncomment to mount pdwfs sources folder into the container (for debug/development)
# a modification is needed in the Dockerfile as well
DOCKER_RUN_OPT = -v $(shell pwd)/../../:/pdwfs

# uncomment to allow stracing in docker (for debug)
#DOCKER_RUN_OPT = --security-opt seccomp:unconfined $(DOCKER_RUN_OPT)

build:
	docker build -t pdwfs-base ../base_dockerfile
	docker build -t ior .

run: build
	docker run $(DOCKER_RUN_OPT) -it --rm -p 8888:8888 -v $(shell pwd)/output:/output -v $(shell pwd)/notebook:/notebook --name ior-run ior

connect:
	docker exec -it ior-run bash

