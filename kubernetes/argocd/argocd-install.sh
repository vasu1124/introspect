#!/bin/bash

kubectl create namespace argocd
kapp -a argocd deploy -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
