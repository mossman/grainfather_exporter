FROM debian:bullseye-slim

COPY grainfather_exporter /
ENTRYPOINT /grainfather_exporter
