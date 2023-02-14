FROM golang:1.19-bullseye AS build

WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /grainfather_exporter

FROM gcr.io/distroless/base-debian11
WORKDIR /

COPY --from=build /grainfather_exporter /grainfather_exporter

EXPOSE 9400

USER nonroot:nonroot

ENTRYPOINT ["/grainfather_exporter", "prometheus"]

