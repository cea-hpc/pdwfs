FROM centos:latest 

RUN yum -y update && yum -y install \
	wget \
	gcc \
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

ENV GO111MODULE=on

#fetch and build Redis
RUN mkdir -p /tmp/src/redis && \
	wget -O redis.tar.gz http://download.redis.io/releases/redis-5.0.3.tar.gz && \
	tar xf redis.tar.gz --strip-components=1 -C /tmp/src/redis && \
	rm redis.tar.gz && \
	cd /tmp/src/redis && make -j "$(nproc)" install

RUN rm -rf /tmp/src

# Switch to non-root user
# replace UID and GID with yours to access your files through a mounted volume
RUN groupadd --gid 1010 dev && \
    useradd --uid 1010 --gid dev dev

ENV HOME /home/dev
RUN mkdir -p ${HOME} && chown dev ${HOME}
USER dev
WORKDIR ${HOME}

COPY banner.sh /tmp/
RUN cat /tmp/banner.sh >> .bashrc

CMD bash



