FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY main.go ./
COPY smtp ./smtp
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/go-smtp .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/go-smtp /go-smtp
EXPOSE 8080 25 465 587
USER nonroot:nonroot
ENTRYPOINT ["/go-smtp"]
