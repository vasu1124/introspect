#!/bin/bash

istioctl manifest generate --set profile=default \
  --set values.gateways.istio-ingressgateway.sds.enabled=true \
| kubectl delete -f -
