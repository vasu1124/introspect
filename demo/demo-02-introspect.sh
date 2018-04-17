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
echo "# expose internal service via L4 TCP LoadBalancer"
echo
pe "kubectl edit service introspect"
wait

echo
echo "# expose internal service via L7 Http Ingress"
echo
pe "less introspect/introspect-ingress-ondemand.yaml"
pe "kubectl apply -f introspect/introspect-ingress-ondemand.yaml"
echo
pe "open https://intro.ingress.devx023462.core.shoot.dev.k8s-hana.ondemand.com"
wait

echo
echo "# exposing introspect service via ingress with own domain"
pe "kubectl apply -f introspect/introspect-ingress-actvirtual.yaml"
echo
pe "open https://introspect.k8s.actvirtual.com"

echo
pe "open http://localhost:8001/static/"

echo
echo "# scale the container manually"
pe "kubectl scale --replicas=3 deployment/introspect --record"
echo
echo "# rolling update to v2.0"
pe "kubectl edit deployment introspect --record"
pe "kubectl rollout history deployment introspect"
pe "kubectl rollout undo deployment introspect"
pe "kubectl rollout history deployment introspect"
wait

echo
echo "# creating a CRD and its API extension"
echo
pe "less ../kubernetes/introspect-crd.yaml"
pe "open http://localhost:8001"
pe "open https://introspect.k8s.actvirtual.com/operator"
wait

echo
pe "less ../kubernetes/useless-machine-1.yaml"
pe "kubectl apply -f ../kubernetes/useless-machine-1.yaml"
pe "kubectl apply -f ../kubernetes/useless-machine-2.yaml"

pe "kubectl edit useless useless-machine-1"


echo
echo "delete or ctrl-c"
wait
kubectl delete -f ../kubernetes/all-in-one/
kubectl delete -f introspect/introspect-ingress-actvirtual.yaml
kubectl delete -f introspect/introspect-ingress-ondemand.yaml
kubectl delete -f ../kubernetes/useless-machine-1.yaml
kubectl delete -f ../kubernetes/useless-machine-2.yaml
kubectl delete -f ../kubernetes/introspect-crd.yaml

kill $proxy_job
