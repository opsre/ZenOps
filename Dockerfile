FROM alpine:3.23

LABEL maintainer=eryajf

ENV TZ=Asia/Shanghai
ENV BINARY_NAME=zenops

RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g" /etc/apk/repositories \
    && apk upgrade \
    && apk add bash curl wget alpine-conf busybox-extras tzdata \
    && apk del alpine-conf && rm -rf /var/cache/*

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY config.example.yaml ./config.yaml
COPY bin/${BINARY_NAME}_${TARGETOS}_${TARGETARCH} ./${BINARY_NAME}

RUN chmod +x ./${BINARY_NAME}

ENTRYPOINT [ "/app/${BINARY_NAME}", "run" ]