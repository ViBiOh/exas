FROM perl:slim

EXPOSE 1080

HEALTHCHECK --retries=10 CMD [ "/exas", "-url", "http://localhost:1080/health" ]
ENTRYPOINT [ "/exas" ]

ARG APP_VERSION
ENV VERSION=${APP_VERSION}

ARG TARGETOS
ARG TARGETARCH

USER 65534
WORKDIR /usr/src/app
COPY exiftool/ .

COPY ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY release/exas_${TARGETOS}_${TARGETARCH} /exas
