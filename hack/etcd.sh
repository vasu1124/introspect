#!/bin/bash
# Run etcd locally for development

docker run -d \
     --name etcd \
     --rm \
     -p 2379:2379 \
     -p 2380:2380 \
     -e ETCD_NAME=etcd0 \
     -e ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379 \
     -e ETCD_ADVERTISE_CLIENT_URLS=http://localhost:2379 \
     -e ETCD_LISTEN_PEER_URLS=http://0.0.0.0:2380 \
     -e ETCD_INITIAL_ADVERTISE_PEER_URLS=http://localhost:2380 \
     -e ETCD_INITIAL_CLUSTER=etcd0=http://localhost:2380 \
     -e ETCD_INITIAL_CLUSTER_STATE=new \
     -e ETCD_INITIAL_CLUSTER_TOKEN=etcd-cluster \
     quay.io/coreos/etcd:v3.6.6

echo "etcd is running on http://localhost:2379"
echo "To stop: docker stop etcd"
echo "To test: docker exec etcd etcdctl --endpoints=http://localhost:2379 endpoint health"
