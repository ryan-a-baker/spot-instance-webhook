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

`helm upgrade --install spot-instance-webhook spot-instance-webhook`


# Testing

In the "test" folder, there are two shell scripts are are intended to be run on a local minikube deployment.  These tests are perfomed by applying known deployments and comparing the deployment that gets created in kubernetes with the expected results.  

It will evaluate the following tests:

Using the Labeled namespace functionality (label-namespace-test.sh):

1) Ensure that node selector/toleration is added to a newly created deployment in a labeled namespace when there previously were no node selectors/tolerations
2) Ensure that node selector/toleration is NOT added to a newly created deployment in a unlabeled namespace
3) Ensure that an update to a deployment in a labeled namespace does not add duplicate node selector/tolerations when mutating
4) Ensure that a deployment in an excluded namespace is not mutated
5) Ensure that a deployment with different node selectors/tolerations appends instead of overwrites when mutatating
6) Ensure that a deployment with a node selector but not a toleration has a toleration added when in a labeled namespace
7) Ensure that a deployment with a toleration but not a node selector has a node selector added when in a labeled namespace
8) Ensure that a deployment with a node selector but not a toleration has a toleration added when in a labeled namespace

Using the "all namespaces" functionality (all-namespace-test.sh):

1) Ensure that node selector/toleration is added to a newly created deployment
1) Ensure that node selector/toleration is NOT added to a newly created deployment when in an excluded namespace

