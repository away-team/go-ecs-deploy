FROM golang:1.10-alpine as builder
WORKDIR /go/src/github.com/away-team/go-ecs-deploy/
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-ecs-deploy src/main.go

FROM alpine:3.7
# Install some common tools often needed during deploys
RUN apk -v --update add ca-certificates bash jq && rm /var/cache/apk/*
COPY templates /templates
WORKDIR /
COPY --from=builder /go/src/github.com/away-team/go-ecs-deploy/go-ecs-deploy .
ENTRYPOINT ["/go-ecs-deploy"]