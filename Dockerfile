FROM golang:latest AS builder

ADD https://github.com/golang/dep/releases/download/v0.5.4/dep-linux-amd64 /usr/bin/dep
RUN chmod +x /usr/bin/dep

WORKDIR /opt/spot-instance-handler
WORKDIR $GOPATH/src/github.com/ryan-a-baker/spot-instance-webhook

COPY ./ $GOPATH/src/github.com/ryan-a-baker/spot-instance-webhook

RUN dep ensure -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /tmp/spot-instance-webhook

FROM alpine:latest
COPY --from=builder /tmp/spot-instance-webhook /
ENTRYPOINT ["/spot-instance-webhook"]

