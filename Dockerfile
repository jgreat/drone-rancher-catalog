FROM alpine:3.6

RUN apk add --no-cache git git-perl bash

ADD ./drone-rancher-catalog /drone-rancher-catalog

CMD [ "/drone-rancher-catalog" ]