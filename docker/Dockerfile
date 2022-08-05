FROM fedbox/builder AS builder

ARG ENV=dev
ARG HOSTNAME=fedbox
ARG STORAGE=all
ARG VERSION

ENV GO111MODULE=on
ENV ENV=${ENV:-dev}
ENV STORAGE=${STORAGE:-all}
ENV VERSION=${VERSION:-}

RUN make ENV=${ENV} STORAGE=${STORAGE} VERSION=${VERSION} all && \
    docker/gen-certs.sh ${HOSTNAME}

FROM gcr.io/distroless/static

ARG PORT=4000
ARG ENV=dev
ARG HOSTNAME=fedbox
ARG STORAGE=all

ENV ENV=${ENV:-dev}
ENV STORAGE_PATH=/storage
ENV HOSTNAME=${HOSTNAME:-fedbox}
ENV LISTEN=:${PORT}
ENV KEY_PATH=/etc/ssl/certs/${HOSTNAME}.key
ENV CERT_PATH=/etc/ssl/certs/${HOSTNAME}.crt
ENV HTTPS=true
ENV STORAGE=${STORAGE:-all}

EXPOSE $PORT

VOLUME /storage
VOLUME /.env

COPY --from=builder /go/src/app/bin/* /bin/
COPY --from=builder /go/src/app/docker/*.key /go/src/app/docker/*.crt /go/src/app/docker/*.pem /etc/ssl/certs/

CMD ["/bin/fedbox"]
