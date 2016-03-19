**Step 1**: Use GVM to install go version 1.4.1+
- `gvm install go1.4.1`
- `vm use go1.4.1`

**Step 2**: Get code
- `go get github.com/huang195/remoteabac`

**Step 3**: Compile and install code
- `godep go install github.com/huang195/remoteabac/cmd/remoteabac`
- `godep go install github.com/huang195/remoteabac/cmd/ruser`

**Step 4**: Run code
- `$GOBIN/remoteabac --address=:8888 --tls-cert-file=cert.pem --tls-private-key-file=key.pem --authorization-policy-file=etcd@http://<ip>:<port>/abac-policy`

**Step 5**: Add/delete user
- `$GOBIN/ruser --authorization-policy-file=etcd@http://<ip>:<port>/abac-policy --type=add --user=alice --namespace=default --readonly=true`
- `$GOBIN/ruser --authorization-policy-file=etcd@http://<ip>:<port>/abac-policy --type=add --user=alice --namespace=alice`
- `$GOBIN/ruser --authorization-policy-file=etcd@http://<ip>:<port>/abac-policy --type=add --user=max --privileged=true`
- `$GOBIN/ruser --authorization-policy-file=etcd@http://<ip>:<port>/abac-policy --type=delete --user=alice`
