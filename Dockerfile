FROM ruby:alpine

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/exas", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/exas" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

VOLUME /tmp

ARG TARGETOS
ARG TARGETARCH

USER 405
WORKDIR /usr/src/app
COPY exiftool/ .

COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY release/exas_${TARGETOS}_${TARGETARCH} /exas
