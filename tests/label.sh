#!/bin/zsh

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\e[0;33m'
NC='\033[0m' # No Color

log() { printf '%s %s\n' "--->" "$1"; }
sublog() { printf '  %s %s ... ' "|-->" "$1"; }
ok () { printf "[${GREEN}OK${NC}]\n"; }
fail() { printf "[${RED}FAIL${NC}]\n"; }
error() { printf "[${RED}Error${NC}] $1\n" && exit 1; }
info() { printf "[${YELLOW}Info${NC}] $1\n"; }
fatal() { fail && error "$1"; }


kubectl create namespace test-unlabeled
kubectl create namespace test-labeled
kubectl label namespace test-labeled spot-deploy=enabled
kubectl apply -f sleep-empty.yaml -n test-unlabeled
kubectl apply -f sleep-empty.yaml -n test-labeled

sleep 15

info "This SHOULD NOT tolerations or node selector"
kubectl get deployment -o yaml sleep -n test-unlabeled
info "This SHOULD have tolerations or node selector"
kubectl get deployment -o yaml sleep -n test-labeled

# When an update is run, it's possible to get a duplicate
# Check to see if we get duplicate tolerations
kubectl label deployment sleep -n test-unlabeled test=test
kubectl label deployment sleep -n test-labeled test=test

sleep 15 

info "This SHOULD NOT tolerations or node selector"
kubectl get deployment -o yaml sleep -n test-unlabeled
info "Check for duplicate tolerations"
kubectl get deployment -o yaml sleep -n test-labeled

tolerations=$(kubectl get deployment -o json sleep -n test-labeled | jq '.spec.template.spec.tolerations | length')
echo $tolerations 
if [ $tolerations -gt 1 ];
then
    echo Error
    echo "Toleration count didn't match expected 1: $tolerations"
fi;

kubectl delete namespace test-unlabeled test-labeled







