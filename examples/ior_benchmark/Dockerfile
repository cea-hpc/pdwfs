FROM pdwfs-base

RUN yum -y update && yum -y install \
	python-devel \
	zlib-devel; \
	yum clean all

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


# Jupyter, matplotlib and pandas
RUN wget -O get-pip.py 'https://bootstrap.pypa.io/get-pip.py' && \
	python get-pip.py && \
	python -m pip install jupyter matplotlib pandas

EXPOSE 8888


# Switch to non-root user
# replace UID and GID with yours to access your files through a mounted volume
RUN groupadd --gid 1010 rebels && \
    useradd --uid 1010 --gid rebels luke

USER luke
ENV HOME /home/luke
WORKDIR ${HOME}
RUN mkdir -p ${HOME}/run

RUN git clone 'https://github.com/cea-hpc/pdwfs' && \
	cd pdwfs && make PREFIX=${HOME}/opt/pdwfs install

# pdwfs bin will be first search in /pdwfs/build which is a local build directory (on the host, not in the container, /pdwfs is a mounted volume)
# if no bin is found, it will look into the container installed version checked out from the GitHub repo
ENV PATH "/pdwfs/build/bin:${HOME}/opt/pdwfs/bin:${PATH}"

COPY banner.sh /tmp/
RUN cat /tmp/banner.sh >> ${HOME}/.bashrc

COPY --chown=luke:rebels jupyter_notebook_config.py ${HOME}/.jupyter/
COPY --chown=luke:rebels ior_script.jinja2 .
COPY --chown=luke:rebels utils.py .
COPY --chown=luke:rebels bench.py .
COPY --chown=luke:rebels notebook/ior_example.ipynb .

CMD bash



