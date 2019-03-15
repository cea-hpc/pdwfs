FROM pdwfs-base

USER root

# Download, build HDF5 and IOR, install in /usr/local

RUN wget -O hdf5.tar.gz 'http://www.hdfgroup.org/ftp/HDF5/releases/hdf5-1.10/hdf5-1.10.5/src/hdf5-1.10.5.tar.gz' && \ 
	mkdir hdf5 && tar xf hdf5.tar.gz --strip-components=1 -C hdf5 && \
	cd hdf5 && ./configure --prefix=/usr/local --enable-parallel && \
	make -j "$(nproc)" install && \
	cd ../ && rm -rf hdf5/ hdf5.tar.gz

RUN git clone 'https://github.com/hpc/ior' && \ 
	cd ior && ./bootstrap && ./configure --with-hdf5 --prefix=/usr/local && \
	make -j "$(nproc)" install && \
	cd .. && rm -rf ior/


USER luke

# Clone, build and install pdwfs in user space

RUN mkdir -p ${HOME}/src && cd ${HOME}/src && \
	git clone 'https://github.com/cea-hpc/pdwfs' && \
	cd pdwfs && git checkout develop && \
	make PREFIX=${HOME}/opt/pdwfs install

ENV PATH "${HOME}/opt/pdwfs/bin:$PATH"

# uncomment to use pdwfs from a volume mounted on the pdwfs source directory on the host (for debug/development)
# (a modification is needed in the Makefile as well) 
ENV PATH "/pdwfs/build/bin:$PATH"

COPY banner.sh /tmp/
RUN cat /tmp/banner.sh >> ${HOME}/.bashrc

RUN mkdir -p ${HOME}/run
WORKDIR ${HOME}

COPY --chown=luke:rebels notebook/ior_example.ipynb .
COPY --chown=luke:rebels notebook/ior_script.jinja2 .

CMD bash



