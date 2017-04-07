FROM alpine

COPY bin/linux/servicebroker-controller /servicebroker-controller

ENTRYPOINT /servicebroker-controller

