# spot-instance-webhook	

This repo provides a mutating webhook for kubernetes which will automatically inject node selectors and tolerations to deployments to allow pods to run on a node tainted as a spot instance.  In order to take advantage of this, your node pools have to have have the taints and labels setup correctly.  

The node selector that is added is:

```
nodeSelector:
  spot: "true"
```

And the toleration that is added is:

```
tolerations:
- effect: NoSchedule
  key: spot
  operator: Equal
  value: "true"
```

The webhook can be configured to mutate deployments based on one of two ways:

## Labeling a namespace

The first scenario is if you don't want every deployment running on spot instances, but maybe you want your less critical environments.  A good use case here would be to ensure development namespaces run on spot instances, but formal testing or production environments continue to run on long lived nodes.

In this mode, the webhook will inspect each deployment and the namespace that it's in.  If the namespace has the `spot-deploy=enabled` label on it, any deployment (and only deployment, not statuefulsets or dameonsets) will have the nodeSelector and tolerations automatically injectet on deployment creation or update.  

If a namespace is updated with the label after the deployment is initially created, the webhook will inject the nodeSelector and toleration anytime the next update is applied to the deployment configuration.

In order to configure the webhook in this mode, the `mutateAllNamespaces` config value in the Helm chart should be set to false.

## Any Deployment

If you would rather run all namespaces on spot instances except the ones you explicitily define, the `mutateAllNamespaces` flag set to true will mutate any deployment regardless of the namespace.  However, you can exclude namespaces with the `namespacesToExclude` helm chart value.  By default, the webhook will exclude the kube-system and kube-public namespaces, and the chart default values will exclude the default and spot-instance-webhook namespaces. 

# Credit where credit is due

This is webhook was HEAVILY drawn from [Banzia's Cloud example](https://github.com/banzaicloud/admission-webhook-example), which is drawn from [morvencao](https://github.com/morvencao/kube-mutating-webhook-tutorial) example.  There pretty much is just scaffolding that remains from those sources, but it served as the base for this work so  thank you to both of those people/groups for leading the way!

# Deploying

First, we need to create a namespace for the webhook to be deployed in:

`kubectl create namespace spot-instance-webhook`

Create a signed cert using the script from the Istio team.  This will create a secret with the private cert in it

`namespace=spot-instance-webhook ./webhook-create-signed-cert.sh`

Now, get the CA bundle from your current context, so the cert that was signed by the K8S api can be trusted.  This varies depending on the K8S provider you are using:

Minikube: `kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}'`
EKS: `aws eks describe-cluster --name <cluster-name> --query cluster.certificateAuthority.data --region <region>`

This, you'll want to place in your helm values file for the `CABundle:` value (minus the %)

Finally, deploy the chart:

`helm upgrade --install spot-instance-webhook spot-instance-webhook --namespace spot-instance-webhook`

If you wish - you can make sure the pod for the webhook is running:

`kubectl get pods --namespace spot-instance-webhook`

# Troubleshooting

Sometimes things don't work quite how you would like it to.  If that's the case, I've found it helpful to tail the logs of the spot-instance-webhook pods.  If it doesn't look like the webhook is being called, take a look at the logs of the K8S API server.  Typically, this is because 

# Testing

In the "test" folder, there are two shell scripts are are intended to be run on a local minikube deployment.  These tests are perfomed by applying known deployments and comparing the deployment that gets created in kubernetes with the expected results.  Please note, these tests can be someone destructive, so please be sure to run them only in a dev environment or minikube.

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

