FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /hvr-backend .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /hvr-backend /app/hvr-backend
COPY --from=build /src/data /app/data
EXPOSE 8080
ENTRYPOINT ["/app/hvr-backend"]
