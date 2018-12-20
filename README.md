
go-ecs-deploy
=============

## Overview

A small library to take make deploying to AWS ECS from remote systems (like CI servers) a little easier.  It supports deploying both long running services attached to a load balancer as well as one-shot tasks that should run then exit.

### Services
A service is specifically an [ECS Service](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_services.html).  It is expected to start and remain running indefinitely.

### Oneshots
A oneshot service is expected to run and exit.  It uses [ECS RunTask](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_run_task.html) under the hood.  Oneshots are useful for running stand-alone database migration containers, or other ad-hoc one off tasks.

## CLI Tool

The included CLI tool loads a service and task config file, injects the `taskDefinition` and deploys it to ECS.

## Usage
AWS credentials must be available in the usual ways (metadata endpoint, credentials file, environment vars).

**Note:** the CLI tool does not yet support profiles, as the intended place to run is a CI job that most likely has env vars or an instance profile set.

**Deploy a service:**

Use only values in the config file:
```sh
go run src/main.go  -service ../deploy-config-dev/example-service/service.json  -task ../deploy-config-dev/example-service/task.json -type service
```


## Config Examples
A full service config.
```json
{
    "cluster": "dev",
    "serviceName": "test-service",
    "taskDefinition": "",
    "loadBalancers": [
        {
            "targetGroupArn": "arn:aws:elasticloadbalancing:us-east-1:265538938700:targetgroup/dev-example/7b4690956daba80b",
            "containerName": "example-service",
            "containerPort": 8080
        }
    ],
    "desiredCount": 2,
    "clientToken": "<REPLACE ME>",
    "role": "arn:aws:iam::265538938700:role/aws-service-role/ecs.amazonaws.com/AWSServiceRoleForECS",
    "deploymentConfiguration": {
        "maximumPercent": 200,
        "minimumHealthyPercent": 100
    },
    "healthCheckGracePeriodSeconds": 5,
    "schedulingStrategy": "REPLICA"
}
```

A full task config.
```json
{
    "family": "dev-example-service",
    "executionRoleArn": "arn:aws:iam::265538938700:role/services/dev-example20181220204138334200000001",
    "containerDefinitions": [
        {
            "name": "example-service",
            "image": "pbxx/example-service:master-latest",
            "repositoryCredentials": {
                "credentialsParameter": "arn:aws:secretsmanager:us-east-1:265538938700:secret:common/dockerhub_pbxx_read_only-Z3Qpuv"
            },
            "cpu": 100,
            "memoryReservation": 50,
            "portMappings": [
                {
                    "containerPort": 8080
                }
            ],
            "essential": true,
            "environment": [
                {
                    "name": "ENVIRONMENT",
                    "value": "dev"
                },
                {
                    "name": "SERVICE_NAME",
                    "value": "example"
                }
            ],
            "dockerLabels": {
                "com.datadoghq.ad.logs": "[{\"service\": \"example-service\"}]"
            }
        }
    ]
}
```

**Note** that with these examples it is expected that `clientToken` will be replaced as necessary.  This is an idempotency token and can be set to unix timestamp or similar if idempotency is not important for the deployment.  It is also recommended to replace the `containerDefinition[0].image` with a specific SHA tagged image.

```bash
jq --arg image "$IMAGE" '.containerDefinitions[0].image = $image' task.json > temp_task.json
jq --arg token "$(date +%s)" '.clientToken = $token' service.json > temp_service.json
```

## Notes

Currently this tool assumes only 1 container definition per task:
* the health check waiters only inspect the first container in the service.
* the environment `[]map[string]string` is only overridden for the first container def

