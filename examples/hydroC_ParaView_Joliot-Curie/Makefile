
# your Joliot-Curie login name can be set here or when calling make: make JOLIOTCURIE_LOGIN=your_username prep
JOLIOTCURIE_LOGIN=$(shell whoami)

# This target must be run on your local machine with internet access and Joliot-Curie access.
# Currently it fetches a slightly modified version of Hydro (to facilitate build process), not yet pushed to upstream repo
prep:  
	rm -rf tmp/ && \
	mkdir -p tmp && \
	git clone 'https://github.com/JCapul/Hydro' tmp/Hydro && \
	rsync -av --exclude=.git/ ../../../pdwfs $(JOLIOTCURIE_LOGIN)@irene-fr.ccc.cea.fr:~/examples


# This target must be run Joliot-Curie supercomputer to build Hydro (mpicc should be available in PATH)
build: 
	rm -rf bin/ && \
	mkdir -p bin/ && \
	make -C tmp/Hydro/HydroC/HydroC99_2DMpi/Src && \
	install tmp/Hydro/HydroC/HydroC99_2DMpi/Src/hydro -D bin/hydro
