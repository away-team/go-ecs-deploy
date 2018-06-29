
go-ecs-deploy
=============

## Overview

A small library to take make deploying to AWS ECS from remote systems (like CI servers) a little easier.  It supports deploying both long running services attached to a load balancer as well as one-shot tasks that should run then exit.

### Services
A service is specifically an [ECS Service](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_services.html).  It is expected to start and remain running indefinitely.

### Oneshots
A oneshot service is expected to run and exit.  It uses [ECS RunTask](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_run_task.html) under the hood.  Oneshots are useful for running stand-alone database migration containers, or other ad-hoc one off tasks.

## CLI Tool

The included CLI tool loads a config file and a template.  The template file is first executed with the variables in the execution environment, then again with the specified config files.  Values set in the config file will override values set in the environment.

Default templates are located in the `templates/` directory, with some example config files under `example/`

## Usage
AWS credentials must be available in the usual ways (metadata endpoint, credentials file, environment vars).

**Note:** the CLI tool does not yet support profiles, as the intended place to run is a CI job that most likely has env vars or an instance profile set.

**Deploy a service:**

Use only values in the config file:
```sh
go run src/main.go -config example/service-a/service-full.json -template templates/service.json.tpl -type service
```

Set some vars in the environment and the rest in a config file.  This is useful for substituting runtime CI variables that change frequently.
```sh
ImageTag=sometag Image=registry/image go run src/main.go -config example/service-a/service-full.json -template templates/service.json.tpl -type service
```

The above commands will:
* replace any template vars with environment variables first, then the config file.
* register a new task definition
* create a new service or update an existing matching service
    * If the service already exists the service template `InitialCount` will be changed to match the `DesiredCount` in ECS, so as to not undo any manual or auto scaling actions.
* block waiting for the service to be stable 


## Config Examples
A full service config.
```json
{
    "ServiceName": "service-a",
    "Cluster": "some-cluster",
    "SchedulerIAMRoleArn": "arn",
    "TaskIAMRoleArn": "arn",
    "TargetGroupArn": "arn",
    "InitialCount": 2,
    "Image": "registry/container",
    "ImageTag": "xyz",
    "MemoryReservation": 128,
    "CPUReservation": 128,
    "AWSLogsGroupName": "name",
    "AWSLogsRegion": "region",
    "Environment": [
        {
            "name": "ENVIRONMENT",
            "value": "dev"
        }
    ]
}
```


A partial service config that will require the missing vars to be set in the environment.  Note that `Image` and `ImageTag` are omitted.
```json
{
    "ServiceName": "service-a",
    "Cluster": "some-cluster",
    "SchedulerIAMRoleArn": "arn",
    "TaskIAMRoleArn": "arn",
    "TargetGroupArn": "arn",
    "InitialCount": 2,
    "MemoryReservation": 128,
    "CPUReservation": 128,
    "AWSLogsGroupName": "name",
    "AWSLogsRegion": "region",
    "Environment": [
        {
            "name": "ENVIRONMENT",
            "value": "dev"
        }
    ]
}
```

## Included templates

The templates in `templates/` are very opinionated and made for common workflows that we have.  Specifically:

* `templates/migration.json.tpl` for running a container with [golang-/migrate](https://github.com/golang-migrate/migrate).  Note the `command` override.
* `templates/service.json.tpl` for running services with some default settings.

## Notes

Currently this tool assumes only 1 container definition per task:
* the health check waiters only inspect the first container in the service.
* the environment `[]map[string]string` is only overridden for the first container def

