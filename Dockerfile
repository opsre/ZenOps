FROM alpine:3.23

LABEL maintainer=eryajf

ENV TZ=Asia/Shanghai
ENV BINARY_NAME=zenops
ENV GIN_MODE=release

RUN sed -i "s/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g" /etc/apk/repositories \
    && apk upgrade \
    && apk add --no-cache bash curl wget alpine-conf busybox-extras tzdata \
    && apk add --no-cache nodejs npm python3 py3-pip uv \
    && apk del alpine-conf && rm -rf /var/cache/*

ENV YARN_REGISTRY=https://registry.npmmirror.com
ENV NPM_CONFIG_REGISTRY=https://registry.npmmirror.com
ENV UV_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

COPY config.example.yaml ./config.yaml
COPY bin/${BINARY_NAME}_${TARGETOS}_${TARGETARCH} ./${BINARY_NAME}

RUN chmod +x ./${BINARY_NAME}

ENTRYPOINT [ "/app/zenops", "run" ]