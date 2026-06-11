FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY sso-service/go.mod sso-service/go.sum ./sso-service/
COPY profile-service/go.mod profile-service/go.sum ./profile-service/
COPY permissions-service/go.mod permissions-service/go.sum ./permissions-service/
COPY sessions-service/go.mod sessions-service/go.sum ./sessions-service/
RUN cd sso-service && go mod download
COPY profile-service/ ./profile-service/
COPY permissions-service/ ./permissions-service/
COPY sessions-service/ ./sessions-service/
COPY sso-service/ ./sso-service/
RUN cd sso-service && CGO_ENABLED=0 go build -o /server ./cmd/sso
RUN cd sso-service && CGO_ENABLED=0 go build -o /migrator ./cmd/migrator

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /server /server
COPY --from=builder /migrator /migrator
COPY sso-service/migrations /migrations
COPY sso-service/config/config_docker.yaml /app/config.yaml
EXPOSE 44044
CMD ["/server"]
