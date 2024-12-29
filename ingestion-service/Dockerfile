FROM golang:1.23 AS build

WORKDIR /go/src/app
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/app

FROM scratch

EXPOSE 8888
COPY --from=build /go/bin/app /
ENTRYPOINT ["/app", "8888"]
