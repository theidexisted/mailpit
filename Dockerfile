FROM alpine:latest
ADD ./mailpit /usr/bin
RUN chmod +x /usr/bin/mailpit
ENTRYPOINT ["/usr/bin/mailpit"]
