FROM golang:1.20 AS build

LABEL maintainer="edoshor@gmail.com"

WORKDIR /build
COPY . .
RUN go build

FROM alpine

WORKDIR /app
COPY ./filer_storage.conf /etc/
COPY --from=build /build/filer-backend .

CMD ["./filer-backend"]