FROM golang:1.22.1-alpine as build

RUN mkdir /app

WORKDIR /app

COPY ./ /app

RUN go mod tidy

RUN go build -o daarul_mukhtarin

EXPOSE 80

CMD [ "./daarul_mukhtarin" ]