{
    "service": {
        "cluster": "{{ .Cluster }}",
        "serviceName": "{{ .ServiceName }}",
        "loadBalancers": [
            {
                "targetGroupArn": "{{ .TargetGroupArn }}",
                "containerName": "{{ .ServiceName }}",
                "containerPort": 8080
            }
        ],
        "desiredCount": {{ .InitialCount }},
        "role": "{{ .SchedulerIAMRoleArn }}",
        "deploymentConfiguration": {
            "maximumPercent": 200,
            "minimumHealthyPercent": 100
        }
    },
    "task": {
        "family": "{{ .TaskFamily }}",
        "taskRoleArn": "{{ .TaskIAMRoleArn }}",
        "containerDefinitions": [
        {
            "name": "{{ .ServiceName }}",
            "image": "{{ .Image}}:{{ .ImageTag }}",
            "essential": true,
            "memoryReservation": {{ .MemoryReservation }},
            "cpu": {{ .CPUReservation }},
            "ulimits": [
                {
                  "name": "nofile",
                  "softLimit": 20000,
                  "hardLimit": 20000
                }
            ],
            "portMappings": [
            {
                "containerPort": 8080,
                "hostPort": 0
            }
            ],
            "logConfiguration": {
                "logDriver": "awslogs",
                "options": {
                    "awslogs-group": "{{ .AWSLogsGroupName }}",
                    "awslogs-region": "{{ .AWSLogsRegion }}",
                    "awslogs-stream-prefix": "{{ .ServiceName }}"
                }
            }
        }
        ]
    }
}