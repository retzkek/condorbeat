FROM golang:1.10 as builder
WORKDIR /go/src/github.com/retzkek/condorbeat
COPY . .
RUN go install


FROM opensciencegrid/osg-wn

# install condor
RUN curl -OL http://research.cs.wisc.edu/htcondor/yum/RPM-GPG-KEY-HTCondor && \
    rpm --import RPM-GPG-KEY-HTCondor && \
    cd /etc/yum.repos.d && \
    curl -OL http://research.cs.wisc.edu/htcondor/yum/repo.d/htcondor-stable-rhel7.repo && \
    yum -q -y install condor.x86_64

RUN mkdir /condorbeat
WORKDIR /condorbeat
COPY --from=builder /go/bin/condorbeat /go/src/github.com/retzkek/condorbeat/condorbeat.* /condorbeat/

CMD ["./condorbeat","-e","-d","*"]
