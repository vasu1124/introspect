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
kubectl proxy -p 8001 >/dev/null 2>&1 & proxy_job=$!

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


echo "Let's see the custom definition of your new UselessMachine resource"
echo
pe "less ../kubernetes/introspect-crd.yaml"
pe "open http://localhost:8001"
pe "open http://localhost:8001/apis/introspect.actvirtual.com/v1alpha1"
wait

echo
echo "Open the introspect UI"
echo
pe "open http://localhost:8001/api/v1/namespaces/default/services/http:introspect:80/proxy/operator"

echo
echo "View a single resource"
echo
pe "less ../config/samples/uselessmachine-1.yaml"
pe "kubectl create -f ../config/samples/uselessmachine-1.yaml"
pe "kubectl create -f ../config/samples/uselessmachine-2.yaml"
wait

echo


echo "Deep dive into those changes"
echo
p "kubectl get uselessmachines -w"
kubectl get uselessmachines -o custom-columns="NAME:.metadata.name,DESIRED:.spec.desiredState,ACTUAL:.status.actualState,MESSAGE:.status.message" -w & watch_job=$!
wait
kill $watch_job

echo
echo "Lets edit one UselessMachine"
pe "kubectl edit uselessmachines machine-1"
wait

echo
echo "delete or ctrl-c"
wait
kubectl delete -f ../kubernetes/all-in-one/
kubectl delete uselessmachines --all

kill $proxy_job

