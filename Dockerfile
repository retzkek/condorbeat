FROM golang:1.10 as builder
WORKDIR /go/src/github.com/retzkek/condorbeat
COPY . .
RUN go install


FROM retzkek/htcondor:8.6.12

RUN mkdir -p /condorbeat
WORKDIR /condorbeat
COPY --from=builder /go/bin/condorbeat /go/src/github.com/retzkek/condorbeat/condorbeat.* /condorbeat/

CMD ["./condorbeat","-e","-d","*"]
