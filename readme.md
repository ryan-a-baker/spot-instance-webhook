# spot-instance-webhook	

This is HEAVILY drawn from Banzia's Cloud example: https://github.com/banzaicloud/admission-webhook-example	

Which is drawn from https://github.com/morvencao/kube-mutating-webhook-tutorial	

A mutating web hook for kubernetes that allows spot instances to be scheduled on tainted instances.	

dep ensure -v	
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o spot-instance-webhook

# Deploying

First, we need to create a namespace for the webhook to be deployed in:

`kubectl create namespace spot-instance-webhook`

Next, let's set our context in our current cluster to the newly created namespace so when the script runs, it creates it all in the right namespace

`kubectl config set-context <context name> --namespace=spot-instance-webhook`

Create a signed cert using the script from the Istio team.  This will create a secret with the private cert in it

`namespace=spot-instance-webhook ./webhook-create-signed-cert.sh`

Now, get the CA bundle from your current context, so the cert that was signed by the K8S api can be trusted

`kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}'`

This, you'll want to place in your values file for the `CABundle:` value (minus the %)

Finally, deploy the chart:

`helm upgrade --install --namespace spot-instance-webhook spot-instance-webhook`


# Testing

Here are the scenarios we need to test

# Place all deployments on spot instances

## Create new deployment

## Update existing deployment

# Place only label namespaces on spot instances
# Node selector already exists
# Different toleration already exists

## Create new deployment in unlabled namespace
Expected output - no taints or tolerations added
## Create new deployment in a labeled namespace
Expected output - taints and tolerations added
## Create new deployment that already has a taint
## Create new deployment that already has a toleration
## Create a new deployment that has both a taint and toleration

## Update existing deployment in an unlabeled namespace
## Update existing deployment in a labeled namespace

# Sad Path

## Pod is not available



# Current Issues
## Pod must be deployed in default namespace


