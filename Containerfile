FROM scratch

COPY ./backend/cmd/handlat/handlat /handlat

ENTRYPOINT [ "/handlat" ]
