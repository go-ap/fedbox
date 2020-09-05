FROM golang:1.15 as builder

ARG ENV=dev
ARG HOSTNAME=fedbox

ENV GO111MODULE=on
ENV ENV=${ENV:-dev}

ADD ./ /go/src/app

WORKDIR /go/src/app
RUN make ENV=${ENV} all
RUN docker/gen-certs.sh ${HOSTNAME}


FROM gcr.io/distroless/base

ARG PORT=4000
ARG ENV=dev
ARG HOSTNAME=fedbox

ENV ENV=${ENV:-dev}
ENV STORAGE_PATH=/storage
ENV HOSTNAME=${HOSTNAME:-fedbox}
ENV LISTEN=:${PORT}
ENV KEY_PATH=/etc/ssl/certs/${HOSTNAME:-fedbox}.key
ENV CERT_PATH=/etc/ssl/certs/${HOSTNAME:-fedbox}.crt
ENV HTTPS=true
ENV STORAGE=fs

EXPOSE $PORT

VOLUME /storage
VOLUME /.env

COPY --from=builder /go/src/app/bin/* /bin/
COPY --from=builder /go/src/app/*.key /go/src/app/*.crt /go/src/app/*.pem /etc/ssl/certs/

CMD ["/bin/fedbox"]
