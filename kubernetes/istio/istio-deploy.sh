#!/bin/bash

istioctl manifest apply --set profile=default \
  --set values.gateways.istio-ingressgateway.sds.enabled=true \
  --set values.global.k8sIngress.enabled=true \
  --set values.global.k8sIngress.enableHttps=true \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/dnsnames'='*.ingress.example.com' \
  --set values.gateways.istio-ingressgateway.serviceAnnotations.'dns\.gardener\.cloud/ttl'='120' \
  
#  --set values.global.k8sIngress.gatewayName=ingressgateway

