#!/usr/bin/env bash

########################
# include the magic
########################
. $(dirname $0)/demo-magic.sh 


########################
# Configure the options
########################

#
# speed at which to simulate typing. bigger num = faster
#
# TYPE_SPEED=20

#
# custom prompt
#
# see http://www.tldp.org/HOWTO/Bash-Prompt-HOWTO/bash-prompt-escape-sequences.html for escape sequences
#
DEMO_PROMPT="${GREEN}➜ ${CYAN}# "
TYPE_SPEED=10

alias k="kubectl"
shopt -s expand_aliases

k delete -f kubernetes/introspect-config.yaml
k delete -f kubernetes/mongodb-secret.yaml
k delete -f kubernetes/mongodb-pvc.yaml
k delete -f kubernetes/mongodb-deployment.yaml
k delete -f kubernetes/mongodb-service.yaml
k delete -f kubernetes/introspect-deployment.yaml
k delete -f kubernetes/introspect-service.yaml

# hide the evidence
clear

pe "k create -f kubernetes/introspect-config.yaml"
pe "k create -f kubernetes/mongodb-secret.yaml"
p "Let's look at our Cloud Provider: Volumes"
pe "k create -f kubernetes/mongodb-pvc.yaml"
p "Now, let's look at our Cloud Provider again: Volumes"
pe "k create -f kubernetes/mongodb-deployment.yaml"
pe "k create -f kubernetes/mongodb-service.yaml"
pe "k create -f kubernetes/introspect-deployment.yaml"
pe "k create -f kubernetes/introspect-service.yaml"
p "Deployment is running now"
pe "k edit svc introspect"
pe "k describe svc introspect"



