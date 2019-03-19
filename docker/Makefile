
# uncomment to allow stracing in docker (for debug)
#DOCKER_RUN_OPT = --security-opt seccomp:unconfined

build: 
	docker build -t pdwfs .

run: build
	docker run $(DOCKER_RUN_OPT) -it --rm -v $(shell pwd)/..:/home/dev/pdwfs -w /home/dev/pdwfs --name pdwfs-dev pdwfs

clean:
	docker rm $(shell docker ps -qa --no-trunc --filter "status=exited"); \
	docker rmi $(shell docker images --filter "dangling=true" -q --no-trunc)



