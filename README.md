# spot-instance-webhook

This is HEAVILY drawn from Banzia's Cloud example: https://github.com/banzaicloud/admission-webhook-example

Which is drawn from https://github.com/morvencao/kube-mutating-webhook-tutorial

It needs some updating and of course the adjustments to injects taints and node selectors.

A mutating web hook for kubernetes that allows spot instances to be scheduled on tainted instances.

dep ensure -v
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o spot-instance-webhook
