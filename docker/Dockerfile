FROM golang:1.14 as builder

ARG ENV

ENV GO111MODULE=on
ENV ENV=${ENV:-dev}

ADD ./ /go/src/app

WORKDIR /go/src/app
RUN make ENV=${ENV} all

FROM gcr.io/distroless/base

ARG LISTEN
ARG ENV

EXPOSE $LISTEN

VOLUME /storage

ENV ENV $ENV

COPY --from=builder /go/src/app/bin/* /bin/

ADD docker/.env.default /.env

CMD ["/bin/fedbox"]
