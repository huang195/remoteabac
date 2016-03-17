FROM ubuntu

MAINTAINER Hai Huang <haih@us.ibm.com>

RUN mkdir -p /opt/kubernetes

COPY remoteabac /opt/kubernetes/

ENTRYPOINT ["/opt/kubernetes/remoteabac"]
