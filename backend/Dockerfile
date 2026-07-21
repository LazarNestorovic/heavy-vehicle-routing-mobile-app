FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /hvr-backend .

FROM alpine:3.20
COPY --from=build /hvr-backend /hvr-backend
EXPOSE 8080
ENTRYPOINT ["/hvr-backend"]
