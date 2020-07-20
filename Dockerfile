FROM golang:1.13 as builder
WORKDIR /go/src/github.com/retzkek/condorbeat
RUN go get github.com/magefile/mage
COPY . .
RUN mage build


FROM retzkek/htcondor:8.6.12

RUN mkdir -p /condorbeat
RUN useradd condorbeat
COPY --from=builder /go/src/github.com/retzkek/condorbeat/condorbeat* /condorbeat/
RUN chown -R condorbeat:condorbeat /condorbeat
WORKDIR /condorbeat
USER condorbeat

CMD ["./condorbeat","-e","-d","*"]
