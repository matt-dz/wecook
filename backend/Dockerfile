FROM --platform=$BUILDPLATFORM golang:1.25.1-alpine AS build
WORKDIR /app
COPY . .
RUN go build -o ./bin/wecook ./cmd/wecook/main.go

FROM alpine:latest AS final
WORKDIR /app
COPY --from=build /app/bin/wecook /app/wecook
EXPOSE 8080
ENTRYPOINT ["/app/wecook"]

