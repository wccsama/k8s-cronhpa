FROM docker.io/alpine

ADD k8s-cronhpa /usr/local/bin

ENTRYPOINT ["/usr/local/bin/k8s-cronhpa"]