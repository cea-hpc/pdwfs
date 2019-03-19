FROM pdwfs-base

USER root

RUN yum -y update; yum -y install numactl-devel; yum clean all

# Download and install ParaView and FFmpeg in /usr/local

RUN wget -O ParaView.tar.xz 'https://www.paraview.org/paraview-downloads/download.php?submit=Download&version=v5.6&type=binary&os=Linux&downloadFile=ParaView-5.6.0-osmesa-MPI-Linux-64bit.tar.xz' && \ 
	mkdir -p /usr/local/ParaView && \
	tar xf ParaView.tar.xz --strip-components=1 -C /usr/local/ParaView && \
	rm ParaView.tar.xz

ENV PATH "/usr/local/ParaView/bin:$PATH"

RUN wget -O ffmpeg.tar.xz 'https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz' && \
	tar xf ffmpeg.tar.xz --strip-components=1 -C /usr/local/bin && \
	rm ffmpeg.tar.xz


# Clone and build Hydro simulation code and pdwfs in user space

USER luke

RUN mkdir -p ${HOME}/src && cd ${HOME}/src && \
	git clone 'https://github.com/JCapul/Hydro' && \
	make -C Hydro/HydroC/HydroC99_2DMpi/Src && \
	install Hydro/HydroC/HydroC99_2DMpi/Src/hydro -D ${HOME}/opt/hydro/bin/hydro

ENV PATH "${HOME}/opt/hydro/bin:$PATH"

RUN cd ${HOME}/src && \
	git clone 'https://github.com/cea-hpc/pdwfs' && \
	make -C pdwfs PREFIX=${HOME}/opt/pdwfs install

ENV PATH "${HOME}/opt/pdwfs/bin:$PATH"

# uncomment to use pdwfs from a volume mounted on the pdwfs source directory on the host (for debug/development)
# (a modification is needed in the Makefile as well) 
#ENV PATH "/pdwfs/build/bin:$PATH"

COPY banner.sh /tmp/
RUN cat /tmp/banner.sh >> ${HOME}/.bashrc

RUN mkdir -p ${HOME}/run
WORKDIR ${HOME}/run

COPY --chown=luke:rebels paraview_run.py .
COPY --chown=luke:rebels process_all.py .
COPY --chown=luke:rebels hydro_input.nml .
COPY --chown=luke:rebels run_on_disk.sh .
COPY --chown=luke:rebels run_on_pdwfs.sh .

CMD bash



