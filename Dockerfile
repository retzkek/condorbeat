FROM golang:1.13 as builder
WORKDIR /go/src/github.com/retzkek/condorbeat
RUN go get github.com/magefile/mage
COPY . .
RUN mage build


FROM retzkek/htcondor:8.6.12

RUN mkdir -p /condorbeat
RUN useradd -u 1001 condorbeat
COPY --from=builder /go/src/github.com/retzkek/condorbeat/condorbeat* /condorbeat/
# make working directory owned by root group for openshift
# https://docs.openshift.com/container-platform/3.3/creating_images/guidelines.html#openshift-container-platform-specific-guidelines
RUN chown -R condorbeat:0 /condorbeat
WORKDIR /condorbeat
USER 1001

CMD ["./condorbeat","-e","-d","*"]
