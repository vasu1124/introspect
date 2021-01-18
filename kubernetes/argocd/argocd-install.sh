#!/bin/bash

export domainname="argocd.ingress.example.com"
export issuer="garden"

kubectl create namespace argocd
kapp -a argocd deploy -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml

# please modify deployment argocd-server and add --insecure flag
# ...
# - argocd-server
# - --staticassets
# - /shared/app
# - --insecure <---!

cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1beta1
kind: VirtualService
metadata:
  name: argocd-vs
  namespace: argocd
spec:
  hosts:
  - "${domainname}"
  gateways:
  - istio-system/${issuer}-gw
  http:
  - match:
    - uri:
        regex: /.*
    route:
    - destination:
        port:
          number: 80
        host: argocd-server
EOF
