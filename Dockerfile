FROM alpine:3.3
ADD build /tmp/build
RUN cp /tmp/build/consul-cleaner /usr/local/bin/
