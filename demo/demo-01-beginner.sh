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
clear

echo "# creating a config"
echo
pe "less beginner/myconfig.yaml"
pe "kubectl apply -f beginner/myconfig.yaml"

echo "# creating a secrect"
echo
pe "less beginner/mysecret.yaml"
pe "kubectl apply -f beginner/mysecret.yaml"

echo "# creating a volume"
echo
pe "less beginner/myclaim.yaml"
pe "kubectl apply -f beginner/myclaim.yaml"

echo "# creating a pod"
echo
pe "less beginner/mypod.yaml"
pe "kubectl apply -f beginner/mypod.yaml"

echo "# exec interactive shell in my pod"
pe "kubectl exec -it mypod -- sh"


echo
echo "delete or ctrl-c"
wait
kubectl delete -f beginner/mypod.yaml
kubectl delete -f beginner/myclaim.yaml
kubectl delete -f beginner/mysecret.yaml
kubectl delete -f beginner/myconfig.yaml
