# Copyright 2013-2019 Lawrence Livermore National Security, LLC and other
# Spack Project Developers. See the top-level COPYRIGHT file for details.
#
# SPDX-License-Identifier: (Apache-2.0 OR MIT)


from spack import *


class Pdwfs(MakefilePackage):
    """
    pdwfs is an open source (Apache 2.0 licensed), preload library implementing 
    a distributed in-memory filesystem in user space suitable for intercepting 
    bulk I/O workloads typical of HPC simulations. It is using Redis as the 
    backend memory store.

    pdwfs (with Redis) provides a lightweight infrastructure to execute HPC simulation
    workflows in transit, i.e. without writing/reading any intermediate data to/from
    a (parallel) filesystem.

    pdwfs is written in Go and C and runs on Linux systems only.
    """

    homepage = "https://github.com/cea-hpc/pdwfs"
    url      = "https://github.com/cea-hpc/pdwfs/archive/v0.1.2.tar.gz"
    git      = "https://github.com/cea-hpc/pdwfs.git"

    version('develop', branch='develop')
    version('0.1.2', sha256='78336ee06985d6ffa7a5e13ecb368cd0f39bcaeb84f99d54337823bce1eba371')

    depends_on('go@1.11:', type='build')
    depends_on('redis', type='run')

    @property
    def install_targets(self):
        return [
            'PREFIX={0}'.format(self.spec.prefix),
            'install'
        ]