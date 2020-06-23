FROM golang:1.13 as builder
WORKDIR /go/src/github.com/retzkek/condorbeat
RUN go get github.com/magefile/mage
COPY . .
RUN mage build


FROM retzkek/htcondor:8.6.12

RUN mkdir -p /condorbeat
WORKDIR /condorbeat
COPY --from=builder /go/src/github.com/retzkek/condorbeat/condorbeat* /condorbeat/

CMD ["./condorbeat","-e","-d","*"]
