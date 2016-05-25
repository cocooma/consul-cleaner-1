FROM alpine:3.3
ADD build/consul-cleaner /usr/local/bin/consul-cleaner
RUN apk add --update ca-certificates
