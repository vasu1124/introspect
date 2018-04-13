#!/bin/bash
dir="${0%/*}"
cd ..
kubectl proxy --www=./kubernetes/k8s-visualizer/src -p 8001
