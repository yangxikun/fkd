FROM alpine:latest
RUN apk add --no-cache tzdata
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
COPY fkd /opt/fkd
CMD ["-logtostderr"]
ENTRYPOINT ["/opt/fkd"]