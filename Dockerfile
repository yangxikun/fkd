FROM alpine:latest
COPY fkd /opt/fkd
CMD ["-logtostderr"]
ENTRYPOINT ["/opt/fkd"]