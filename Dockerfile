FROM debian:bullseye-slim

RUN apt-get -y update ; apt-get install -y curl

COPY grainfather_exporter /
CMD ["/grainfather_exporter", "prometheus"]

