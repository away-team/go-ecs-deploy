{
    "task": {
        "family": "{{ .TaskFamily }}",
        "taskRoleArn": "{{ .TaskIAMRoleArn }}",
        "containerDefinitions": [
            {
                "name": "migration",
                "image": "{{ .Image }}:{{ .ImageTag }}",
                "memory": 100,
                "essential": true,
                "logConfiguration": {
                    "logDriver": "awslogs",
                    "options": {
                        "awslogs-group": "{{ .AWSLogsGroupName }}",
                        "awslogs-region": "{{ .AWSLogsRegion }}",
                        "awslogs-stream-prefix": "{{ .ServiceName }}"
                    }
                },
                "command": ["-env", "{{ .Cluster }}", "-service", "{{ .ServiceName }}", "-path", "/migration", "up"]
            }
        ]
    }
}