#!/bin/bash

export domainname="*.ingress.example.com"
export issuer="garden"

istioctl manifest apply --set profile=default \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/dnsnames'=${domainname} \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/ttl'='120' \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/class'='garden' \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'cert\.gardener\.cloud/issuer'=${issuer} \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'cert\.gardener\.cloud/secretname'='wildcard-tls' \
  --set values.gateways.istio-ingressgateway.sds.enabled=true 

#  --set values.global.k8sIngress.gatewayName=ingressgateway

cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1beta1
kind: Gateway
metadata:
  name: ${issuer}_gw
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
