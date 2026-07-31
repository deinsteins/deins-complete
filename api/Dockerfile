FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o /out/deinscomplete-api ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/deinscomplete-migrate ./cmd/migrate \
 && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/deinscomplete-admin ./cmd/admin

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/deinscomplete-api /deinscomplete-api
COPY --from=builder /out/deinscomplete-migrate /deinscomplete-migrate
COPY --from=builder /out/deinscomplete-admin /deinscomplete-admin
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/deinscomplete-api"]
