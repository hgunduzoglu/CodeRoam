FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY services/control-plane/go.mod ./
COPY services/control-plane/ ./
RUN CGO_ENABLED=0 go build -trimpath -o /out/coderoam-api ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/coderoam-api /usr/local/bin/coderoam-api
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/coderoam-api"]
