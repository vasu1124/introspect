#!/usr/bin/env bash

########################
# include the magic
########################
dir="${0%/*}"
. $dir/demo-magic.sh


########################
# Configure the options
########################

#
# speed at which to simulate typing. bigger num = faster
#
TYPE_SPEED=30

#
# custom prompt
#
# see http://www.tldp.org/HOWTO/Bash-Prompt-HOWTO/bash-prompt-escape-sequences.html for escape sequences
#
DEMO_PROMPT="${GREEN}➜ ${CYAN}\W "

# hide the evidence
cd $dir
kubectl proxy --www=../kubernetes/k8s-visualizer/src -p 8001 >/dev/null 2>&1 & proxy_job=$!

cleanup() {
  kill $proxy_job
  exit
}

trap cleanup INT
clear

echo "# start introspect"
echo
pe "ls ../kubernetes/all-in-one/"
pe "kubectl apply -f ../kubernetes/all-in-one/"


echo
echo "# expose internal service via L7 Http Ingress"
echo
pe "less introspect/introspect-ingress-ondemand.yaml"
pe "kubectl apply -f introspect/introspect-ingress-ondemand.yaml"
pe "kubectl describe ingress introspect"
echo
pe "open https://intro.ingress.d023462.core.shoot.canary.k8s-hana.ondemand.com"

wait

echo
echo "# creating a CRD and its API extension"
echo
pe "less ../kubernetes/introspect-crd.yaml"
pe "kubectl api-versions"
pe "open http://localhost:8001/apis/introspect.actvirtual.com/v1alpha1"
wait

echo
pe "less ../kubernetes/useless-machine-1.yaml"
pe "kubectl apply -f ../kubernetes/useless-machine-1.yaml"
pe "kubectl apply -f ../kubernetes/useless-machine-2.yaml"

echo "Watch changes"
echo
p "kubectl get uselessmachines -w"
kubectl get uselessmachines -o custom-columns="NAME:.metadata.name,DESIRED:.spec.desiredState,ACTUAL:.status.actualState,MESSAGE:.status.message" -w & watch_job=$!
wait
kill $watch_job

pe "kubectl edit useless useless-machine-1"


echo
echo "delete or ctrl-c"
wait
kubectl delete -f ../kubernetes/all-in-one/
kubectl delete -f introspect/introspect-ingress-ondemand.yaml
kubectl delete uselessmachines --all
kubectl delete -f ../kubernetes/introspect-crd.yaml

kill $proxy_job
