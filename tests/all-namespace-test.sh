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

# Check to make sure that tolerations and selectors get added to a non-excluded namespace

kubectl create namespace test-unlabeled
kubectl apply -f sleep-empty.yaml -n test-unlabeled

# Sleep to make sure K8S has time to create the resource
sleep 5

test-populated test-unlabeled "Resource Creation Mutation on any namespace" toleration-output-1.json nodeSelector-output-1.json

# When an update is run, it's possible to add multiple duplicate tolerations and node selectors,
# so lets test to make sure that doesn't happen
kubectl label deployment sleep -n test-unlabeled test=test

# Sleep to make sure K8S has time to create the resource
sleep 5

test-populated test-unlabeled "Existance of duplication tolerations or selectors after update on labeled namespace" toleration-output-1.json nodeSelector-output-1.json

# Test a namespace that we have excluded to make sure it is not getting
# tolerations.  Even if it's labeled, it should be excluded from the mutating webhook.

kubectl label namespace default spot-deploy=enabled
kubectl apply -f sleep-empty.yaml -n default
sleep 5
test-unpopulated default "Excluded namespace is skipped"
# This is outside the normal namespace, so lets cleanup the deployment
kubectl delete deployment sleep -n default
kubectl label namespace default spot-deploy-

# Clean up all out testing
#kubectl delete namespace test-unlabeled







