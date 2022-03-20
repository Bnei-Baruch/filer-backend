FROM golang:1.17-alpine3.15 as build

LABEL maintainer="edoshor@gmail.com"

WORKDIR /build
COPY . .
RUN go build

FROM alpine:3.15

WORKDIR /app
COPY ./filer_storage.conf /etc/
COPY --from=build /build/filer-backend .

CMD ["./filer-backend"]