# Changelog

## vO.2.1

- Fix a memory corruption issue by passing a copy of pathnames from C to Go [2f9e3db]
- Update README.md [f893f1e]
- Add file metadata caching in client [0a098e0]

## v0.2.0

- Add Spack package for pdwfs [b6cb1f2]
- Update README.md [1e062df]
- Add an example for using pdwfs with SLURM job scheduler [b2839c1]
- Update SLURM helpers scripts [3aca9ae]
- Increase default stripe size to 50MB [9907704]
- Merge branch 'feature/custom-redigo-client' into develop [6705d0c]
- Fix performance issues on write and read [8a54135]
- Optimize write with Redis Set cmd if whole buffer is to be written (faster) [bd34304]
- Refactor tests to improve isolation and avoid need for running Redis beforehand [9ea7596]
- Refactor the C layer to move all "triage" into C layer and minimize CGo cross-layer calls [1f53f0c]

## v0.1.2

- Refactor the inode layer to remove a lock [0b573de]
- Add redis address and blocksize conf with env var [31292e9]
- Update examples [55eb7aa]

## v0.1.1

- Fix errno management from Go to C layer [054dd09]
- Add example with IOR benchmark [6e23b41]
- Update README.md [a740d70]
- Add example workflow with HydroC code and ParaView [88b72c1]
- Add a development docker environment [12af2a6]

## v0.1.0

- Hello github !