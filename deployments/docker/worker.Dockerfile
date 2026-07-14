FROM golang:1.26-alpine AS build
WORKDIR /src
COPY services/worker/go.mod ./
COPY services/worker/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/coderoam-worker ./cmd/worker

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coderoam-worker /usr/local/bin/coderoam-worker
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/coderoam-worker"]
