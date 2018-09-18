Dockerfile and config for running htcondor and a go environment.

Based off https://github.com/andypohl/htcondor-docker, but modified for Debian. Original README follows.

# htcondor-docker
Dockerfile for building a Docker image with the latest "personal" HTCondor on CentOS 7.  To run the test:
```
me@laptop$ docker run -d -h htcondor --name htcondor andypohl/htcondor
63929dc053607d52071c99933520ff0bbda887e125c4bd5866ae976283626b5a
me@laptop$ docker exec -ti -u 1000:1000 htcondor bash
[submitter@htcondor submit]$ cd ../example/
[submitter@htcondor example]$ condor_status
Name           OpSys      Arch   State     Activity LoadAv Mem   ActvtyTime

slot1@htcondor LINUX      X86_64 Unclaimed Idle      0.000  999  0+00:00:03
slot2@htcondor LINUX      X86_64 Unclaimed Idle      0.000  999  0+00:00:03

                     Machines Owner Claimed Unclaimed Matched Preempting  Drain

        X86_64/LINUX        2     0       0         2       0          0      0

               Total        2     0       0         2       0          0      0
[submitter@htcondor example]$ condor_submit hello.sub
Submitting job(s).
1 job(s) submitted to cluster 1.
```
