## Job

The workflow is specified in two similar SLURM jobs: 
- ```job_without_pdwfs.sh```: specifies the workflow without pdwfs, ie using regular files on the scratch file system
- ```job_with_pdwfs.sh```: same as job.sh but using pdwfs to store files

To execute a job in interactive mode, use the provided ```run_interactive.sh``` script after commenting out the line for the job you don't want to run:
```bash
$ ./run_interactive.sh
```

To execute a job in batch mode:
```bash
$ ccc_msub ./job.sh
```
or
```bash
$ ccc_msub ./job_with_pdwfs.sh
```

