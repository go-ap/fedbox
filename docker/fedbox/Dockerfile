ARG ENV
ARG HOSTNAME
FROM ${HOSTNAME}/builder:${ENV} as builder

FROM gcr.io/distroless/base

ARG LISTEN
EXPOSE $LISTEN

VOLUME /storage

ENV ENV $ENV

COPY --from=builder /go/src/app/bin/fedbox /bin/fedbox
COPY --from=builder /go/src/app/bin/ctl /bin/ctl

COPY .env* ./

CMD ["/bin/fedbox"]

