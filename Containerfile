FROM alpine:3.23 AS certificates

RUN apk add --no-cache ca-certificates

FROM scratch

COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY ./backend/cmd/handlat/handlat /handlat

ENTRYPOINT [ "/handlat" ]
