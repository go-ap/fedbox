FROM golang:1.15 as builder

ARG ENV=dev
ARG HOSTNAME=fedbox
ARG STORAGE=all

ENV GO111MODULE=on
ENV ENV=${ENV:-dev}
ENV STORAGE=${STORAGE:-all}

ADD ./ /go/src/app

WORKDIR /go/src/app
RUN make ENV=${ENV} STORAGE=${STORAGE} clean all
RUN docker/gen-certs.sh fedbox

FROM gcr.io/distroless/base

ARG PORT=4000
ARG ENV=dev
ARG HOSTNAME=fedbox
ARG STORAGE=all

ENV ENV=${ENV:-dev}
ENV STORAGE_PATH=/storage
ENV HOSTNAME=${HOSTNAME:-fedbox}
ENV LISTEN=:${PORT}
ENV KEY_PATH=/etc/ssl/certs/fedbox.key
ENV CERT_PATH=/etc/ssl/certs/fedbox.crt
ENV HTTPS=true
ENV STORAGE=${STORAGE:-all}

EXPOSE $PORT

VOLUME /storage
VOLUME /.env

COPY --from=builder /go/src/app/bin/* /bin/
COPY --from=builder /go/src/app/*.key /go/src/app/*.crt /go/src/app/*.pem /etc/ssl/certs/

CMD ["/bin/fedbox"]
