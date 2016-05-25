FROM alpine:3.3
ADD build /tmp/build
RUN cp /tmp/build/* /usr/local/bin/
