#!/bin/bash

export domainname="*.ingress.example.com"

istioctl manifest apply --set profile=default \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/dnsnames'=${domainname} \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/ttl'='120' \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/class'='garden' \
  --set values.gateways.istio-ingressgateway.sds.enabled=true 

# workaround: with istioctl 1.5.2 it seems you need to add k8sIngress incrementally.
# this workaround breaks if any component is restarted.
sleep 30

istioctl manifest apply --set profile=default \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/dnsnames'=${domainname} \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/ttl'='120' \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/class'='garden' \
  --set values.gateways.istio-ingressgateway.sds.enabled=true \
  --set values.global.k8sIngress.enabled=true \
  --set values.global.k8sIngress.enableHttps=true

#  --set values.global.k8sIngress.gatewayName=ingressgateway