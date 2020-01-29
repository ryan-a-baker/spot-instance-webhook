FROM alpine:latest

ADD spot-instance-webhook /spot-instance-webhook
ENTRYPOINT ["./spot-instance-webhook"]