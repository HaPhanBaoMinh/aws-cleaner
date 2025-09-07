# syntax=docker/dockerfile:1

########## BUILD ##########
FROM --platform=$BUILDPLATFORM golang:1.22-alpine AS build
WORKDIR /src

# certs cho HTTPS, cài ở build stage
RUN apk add --no-cache ca-certificates

ENV CGO_ENABLED=0 GOOS=linux GO111MODULE=on

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/aws-cleaner ./


########## RUN ##########
FROM scratch
WORKDIR /

# copy CA certs đã cài ở build stage
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# copy binary
COPY --from=build /out/aws-cleaner /aws-cleaner

USER 10001:10001
ENTRYPOINT ["/aws-cleaner"]

