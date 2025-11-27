FROM --platform=$BUILDPLATFORM golang:1.25.1-alpine AS build
RUN apk update && apk add make golangci-lint perl-dev
RUN go install github.com/swaggo/swag/cmd/swag@latest
ENV PG_FORMAT_VERSION=5.8
RUN wget https://github.com/darold/pgFormatter/archive/refs/tags/v${PG_FORMAT_VERSION}.tar.gz

RUN tar xzf v${PG_FORMAT_VERSION}.tar.gz
RUN cd pgFormatter-${PG_FORMAT_VERSION} && \
	perl Makefile.PL && \
	make && \
	make install
RUN rm -rf v${PG_FORMAT_VERSION}.tar.gz && \
	rm -rf pgFormatter-${PG_FORMAT_VERSION}
RUN pg_format --version

ENV GOLANGCI_LINT_VERSION=v2.5.0
RUN wget -O- -nv https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s ${GOLANGCI_LINT_VERSION}
RUN golangci-lint --version

WORKDIR /app
COPY . .
RUN make build

FROM alpine:latest AS final
WORKDIR /app
COPY --from=build /app/bin/wecook /app/wecook
EXPOSE 8080
ENTRYPOINT ["/app/wecook"]

