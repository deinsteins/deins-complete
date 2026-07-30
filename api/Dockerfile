FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=dev
ARG COMMIT=unknown
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.Version=${VERSION} -X main.Commit=${COMMIT}" -o /out/deinscomplete-api ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/deinscomplete-api /deinscomplete-api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/deinscomplete-api"]
