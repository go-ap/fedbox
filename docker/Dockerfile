FROM golang:1.14 as builder

ARG ENV=dev
ARG HOSTNAME=fedbox

ENV GO111MODULE=on
ENV ENV=${ENV:-dev}

ADD ./ /go/src/app

WORKDIR /go/src/app
RUN make ENV=${ENV} all

FROM gcr.io/distroless/base

ARG PORT=4000
ARG ENV=dev

ENV ENV=${ENV}
ENV STORAGE_PATH=/storage
ENV HTTPS=true
ENV KEY_PATH=/etc/ssl/certs/fedbox.key
ENV CERT_PATH=/etc/ssl/certs/fedbox.crt
ENV STORAGE=fs
ENV LISTEN=:${PORT}
ENV HOSTNAME=fedbox

EXPOSE $PORT

VOLUME /storage
VOLUME /.env

COPY --from=builder /go/src/app/bin/* /bin/
ADD ./docker/*.key /etc/ssl/certs/fedbox.key
ADD ./docker/*.pem /etc/ssl/certs/fedbox.pem

CMD ["/bin/fedbox"]
