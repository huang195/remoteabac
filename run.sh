#!/bin/bash

# standalone mode
#$GOPATH/bin/remoteabac --address=:8888 --tls-cert-file=apiserver.pem --tls-private-key-file=apiserver-key.pem --authorization-policy-file=etcd@http://k8sm1:4001/abac-policy

# containerized mode
docker run -p 8888:8888 -v `pwd`:/tmp haih/remoteabac --address=:8888 --tls-cert-file=/tmp/apiserver.pem --tls-private-key-file=/tmp/apiserver-key.pem --authorization-policy-file=etcd@http://10.143.100.209:4001/abac-policy
