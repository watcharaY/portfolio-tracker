FROM golang:1.27 AS build
# The image pins GOTOOLCHAIN=local; auto lets go.mod's toolchain line pick the exact Go version.
ENV GOTOOLCHAIN=auto CGO_ENABLED=0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/server /server
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=6s --start-period=10s --retries=3 CMD ["/server", "healthcheck"]
ENTRYPOINT ["/server"]
