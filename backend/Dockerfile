FROM --platform=$BUILDPLATFORM golang:1.25.1-alpine AS build
WORKDIR /app
COPY . .
RUN apk update && apk add make
RUN make build

FROM alpine:latest AS final
WORKDIR /app
COPY --from=build /app/bin/wecook /app/wecook
EXPOSE 8080
ENTRYPOINT ["/app/wecook"]

