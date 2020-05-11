#!/bin/bash

test-populated () {
    printf "${YELLOW}Test $2: ${NC}"
    diff <(kubectl get deployment sleep -o json  -n $1 | jq '.spec.template.spec.tolerations' | jq -S .) <(jq -S . $3)
    if [ $? != 0 ]
    then
        printf "${RED}Failed${NC}\n"
        error-out
    fi;

    diff <(kubectl get deployments sleep -o json -n $1  | jq '.spec.template.spec.nodeSelector' | jq -S .) <(jq -S . $4)
    if [ $? != 0 ]
    then
        printf "${RED}Failed${NC}\n"
        error-out
    fi;  
    
    printf "${GREEN}Passed${NC}\n"
}

test-unpopulated () {
    printf "${YELLOW}Test $2: ${NC}"
    if [ $(kubectl get deployment -o json sleep -n $1 | jq '.spec.template.spec.tolerations | length') != 0 ]
    then
        printf "${RED}Failed${NC}\n"
        error-out
    fi;

    if [ $(kubectl get deployment -o json sleep -n $1 | jq '.spec.template.spec.nodeSelector | length') != 0 ]
    then
        printf "${RED}Failed${NC}\n"
        error-out
    fi;

    printf "${GREEN}Passed${NC}\n"
}

error-out (){
    echo "Error occured during testing.  Leaving the deployments for troubleshooting"
    echo "When finished troubleshooting, delete namespaces before re-running using the following command"
    echo "kubectl delete namespace test-unlabeled test-labeled"
    exit 1
}

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\e[0;33m'
NC='\033[0m' # No Color

# First we'll test to see if freshly created resources in 
# a labeled gets the mutations, as well as no mutiations
# in an unlabled namespace

kubectl create namespace test-unlabeled
kubectl create namespace test-labeled-1
kubectl label namespace test-labeled-1 spot-deploy=enabled
kubectl apply -f sleep-empty.yaml -n test-unlabeled
kubectl apply -f sleep-empty.yaml -n test-labeled-1

# Sleep to make sure K8S has time to create the resource
sleep 5

test-populated test-labeled-1 "Resource Creation Mutation on labeled namespace with no tolerations or selectors" toleration-output-1.json nodeSelector-output-1.json
test-unpopulated test-unlabeled "Resource Creation Mutation on unlabeled namespace with no tolerations or selectors"

# When an update is run, it's possible to add multiple duplicate tolerations and node selectors,
# so lets test to make sure that doesn't happen
kubectl label deployment sleep -n test-unlabeled test=test
kubectl label deployment sleep -n test-labeled-1 test=test

# Sleep to make sure K8S has time to create the resource
sleep 5

test-populated test-labeled-1 "Existance of duplication tolerations or selectors after update on labeled namespace" toleration-output-1.json nodeSelector-output-1.json
test-unpopulated test-unlabeled "Existance of duplication tolerations or selectors after update on unlabeled namespace"

# sleep 5

# # Check to see if a deployment after a label is applied is updated with tolerations

# Let's make sure the tolerations and selectors are added when a namespace has the label
# added after the resources are initially created
kubectl create namespace test-labeled-2
kubectl apply -f sleep-empty.yaml -n test-labeled-2
# As a sanity check, let's go ahead and make sure the tolerations were not added
test-unpopulated test-labeled-2 "Stage update resource test"

kubectl label namespace test-labeled-2 spot-deploy=enabled
kubectl label deployment sleep -n test-labeled-2 test=test
sleep 5
test-populated test-labeled-2 "Resource Update Mutation on labeled namespace" toleration-output-1.json nodeSelector-output-1.json

# Test a namespace that we have excluded to make sure it is not getting
# tolerations.  Even if it's labeled, it should be excluded from the mutating webhook.

kubectl label namespace default spot-deploy=enabled
kubectl apply -f sleep-empty.yaml -n default
sleep 5
test-unpopulated default "Excluded namespace is skipped"
# This is outside the normal namespace, so lets cleanup the deployment
kubectl delete deployment sleep -n default
kubectl label namespace default spot-deploy-

# We should check if that if a node selector and a toleration already 
# exist, our toleration should be appended, not overwritten

kubectl create namespace test-labeled-3
kubectl label namespace test-labeled-3 spot-deploy=enabled
kubectl apply -f sleep-append-check.yaml -n test-labeled-3
test-populated test-labeled-3 "Ensure existing tolerations and selectors are not overwritten" toleration-output-2.json nodeSelector-output-2.json

# Just for giggles, lets test if only a toleration or selector is in the create, make sure the other is added
kubectl create namespace test-labeled-4
kubectl label namespace test-labeled-4 spot-deploy=enabled
kubectl apply -f sleep-tolerations.yaml -n test-labeled-4
test-populated test-labeled-4 "Ensure node selector is added to a deployment with a toleration but not selector" toleration-output-1.json nodeSelector-output-1.json
kubectl delete deployment sleep -n test-labeled-4
kubectl apply -f sleep-selector.yaml -n test-labeled-4
test-populated test-labeled-4 "Ensure toleration is added to a deployment with a selector but not a toleration" toleration-output-1.json nodeSelector-output-1.json

# Clean up all out testing
kubectl delete namespace test-unlabeled test-labeled-1 test-labeled-2 test-labeled-3 test-labeled-4







