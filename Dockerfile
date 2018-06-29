FROM golang:1.10-alpine as builder
WORKDIR /go/src/github.com/away-team/go-ecs-deploy/
COPY . ./
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o go-ecs-deploy src/main.go

FROM alpine:3.7
# Install some common tools needed during deploys
RUN apk -v --update add \
        bash \
        jq \
        python \
        py-pip \
        groff \
        less \
        mailcap \
        && \
    pip install --upgrade awscli==1.14.5 s3cmd==2.0.1 python-magic && \
    apk -v --purge del py-pip && \
    rm /var/cache/apk/*
COPY templates /templates
WORKDIR /
COPY --from=builder /go/src/github.com/away-team/go-ecs-deploy/go-ecs-deploy .
ENTRYPOINT ["/go-ecs-deploy"]