FROM docker.io/library/alpine:3.23 AS certificates

FROM scratch

# The MCP server needs public CA roots to fetch OAuth client metadata over HTTPS.
COPY --from=certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY ./backend/cmd/handlat/handlat /handlat

ENTRYPOINT [ "/handlat" ]
