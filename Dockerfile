FROM alpine:3.3
ADD build/consul-cleaner /usr/local/bin/consul-cleaner
RUN apk --update add ca-certificates
