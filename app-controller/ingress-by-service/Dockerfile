FROM golang:1.23.0 AS build

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 go build -o ingress-by-service .


FROM alpine:3.18

WORKDIR /app

COPY --from=build /app/ingress-by-service .


CMD [ "./ingress-by-service" ]



