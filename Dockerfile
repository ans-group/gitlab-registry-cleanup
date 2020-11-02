FROM golang:1.15 AS gobuilder
WORKDIR /build
COPY go.mod go.sum /build/
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build

FROM alpine:3.12
RUN apk update && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/*
COPY --from=gobuilder /build/gitlab-registry-cleanup /bin/gitlab-registry-cleanup
ENTRYPOINT ["/bin/gitlab-registry-cleanup"]