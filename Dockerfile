FROM alpine:3.4
RUN apk update && apk add bash ca-certificates git
ADD drone-rancher-catalog /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/drone-rancher-catalog" ]
