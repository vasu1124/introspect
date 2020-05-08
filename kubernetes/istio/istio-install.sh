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