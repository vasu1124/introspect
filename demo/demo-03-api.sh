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

#kubectl proxy -p 8001 >/dev/null 2>&1 & proxy_job=$!
#
# cleanup() {
#   kill $proxy_job
#   exit
# }

# trap cleanup INT
clear

pe "kubectl api-resources"
pe "kubectl explain Node --recursive"

pe "bat ../pkg/operator/useless/config/crd/bases/introspect.actvirtual.com_uselessmachines.yaml"
pe "kubectl apply -f ../pkg/operator/useless/config/crd/bases/introspect.actvirtual.com_uselessmachines.yaml"
pe "kubectl api-resources | grep useless"
pe "kubectl explain UselessMachine --recursive"

wait
kubectl delete -f ../pkg/operator/useless/config/crd/bases/introspect.actvirtual.com_uselessmachines.yaml

