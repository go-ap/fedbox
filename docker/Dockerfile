FROM golang:1.14 as builder

ARG ENV=dev

ENV GO111MODULE=on
ENV ENV=${ENV:-dev}

ADD ./ /go/src/app

WORKDIR /go/src/app
RUN make ENV=${ENV} all

FROM gcr.io/distroless/base

ARG LISTEN=:4000
ARG ENV=dev

EXPOSE $LISTEN

VOLUME /storage

COPY --from=builder /go/src/app/bin/* /bin/

ADD docker/.env.default /.env

CMD ["/bin/fedbox"]
