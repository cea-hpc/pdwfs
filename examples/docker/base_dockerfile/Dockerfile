FROM centos:latest 

RUN yum -y update && yum -y install \
	wget \
	gcc \
	gcc-c++ \
	automake \
	make \
	strace \
	git \
	glib2-devel \
	yum clean all

# Go language
RUN wget -O go.tar.gz 'https://dl.google.com/go/go1.11.5.linux-amd64.tar.gz' && \
	tar xf go.tar.gz -C /usr/local && \
	rm go.tar.gz

ENV PATH "/usr/local/go/bin:$PATH"

# OpenMPI
RUN mkdir -p /tmp/src/openmpi && \
	wget -O openmpi.tar.gz 'https://download.open-mpi.org/release/open-mpi/v2.1/openmpi-2.1.6.tar.gz' && \
	tar xf openmpi.tar.gz --strip-components=1 -C /tmp/src/openmpi && \
	rm openmpi.tar.gz && \
	cd /tmp/src/openmpi && \
	./configure --prefix=/usr/local && make -j "$(nproc)" install

# Redis
RUN mkdir -p /tmp/src/redis && \
	wget -O redis.tar.gz http://download.redis.io/releases/redis-5.0.3.tar.gz && \
	tar xf redis.tar.gz --strip-components=1 -C /tmp/src/redis && \
	rm redis.tar.gz && \
	cd /tmp/src/redis && make PREFIX=/usr/local -j "$(nproc)" install

RUN rm -rf /tmp/src

CMD bash



