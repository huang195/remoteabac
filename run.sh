#!/bin/bash

$GOPATH/bin/remoteabac --address=:8888 --tls-cert-file=apiserver.pem --tls-private-key-file=apiserver-key.pem --authorization-policy-file=etcd@http://k8sm1:4001/abac-policy
