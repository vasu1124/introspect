#!/bin/bash

export domainname="*.ingress.example.com"
export issuer="garden"

cat <<EOF | istioctl install -y -f -
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
spec:
  profile: default
  components:
    ingressGateways:
    - name: istio-ingressgateway
      enabled: true
      k8s:
        serviceAnnotations:
          dns.gardener.cloud/dnsnames: "${domainname}"
          dns.gardener.cloud/ttl: "300" 
          dns.gardener.cloud/class: garden
          cert.gardener.cloud/issuer: "${issuer}"
          cert.gardener.cloud/secretname: wildcard-tls
EOF

cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: ${issuer}-gw
  namespace: istio-system
spec:
  selector:
    istio: ingressgateway #! use istio default ingress gateway
  servers:
  - port:
      number: 80
      name: http
      protocol: HTTP
    tls:
      httpsRedirect: true
    hosts:
    - "${domainname}"
  - port:
      number: 443
      name: https
      protocol: HTTPS
    tls:
      mode: SIMPLE
      credentialName: wildcard-tls
    hosts:
    - "${domainname}"
EOF
