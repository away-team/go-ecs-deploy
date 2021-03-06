package ecsdeploy

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/service/ecs"
)

func (e *ECSDeployer) deployOneshot(taskDef *ecs.RegisterTaskDefinitionInput, count, wait int) error {
	task, err := e.registerTask(taskDef)
	if err != nil {
		return err
	}

	res, err := e.client.RunTask(&ecs.RunTaskInput{
		Cluster:        makeStrPtr(e.cluster),
		TaskDefinition: task.TaskDefinitionArn,
		Count:          makeInt64Ptr(int64(1)),
	})
	if err != nil {
		return err
	}

	if len(res.Failures) > 0 {
		// assume 1 task for now

		var arn string
		if res.Failures[0] != nil {
			arn = *res.Failures[0].Arn
		}

		var reason string
		if res.Failures[0].Reason != nil {
			reason = *res.Failures[0].Reason
		}

		return fmt.Errorf("%s (%s)", arn, reason)
	}

	return e.waitForOneshot(res.Tasks[0].TaskArn, count, wait)
}

// waits for status == STOPPED and exit code == 0
func (e *ECSDeployer) waitForOneshot(taskArn *string, count, wait int) error {
	maxAttempts := count
	delay := time.Duration(wait) * time.Second

	for i := 0; i <= maxAttempts; i++ {

		res, err := e.client.DescribeTasks(&ecs.DescribeTasksInput{
			Cluster: makeStrPtr(e.cluster),
			Tasks: []*string{
				taskArn,
			},
		})
		if err != nil {
			return err
		}

		status := "TEMP"
		exit := 255

		// assume 1 task with 1 container
		if len(res.Tasks) > 0 {
			if len(res.Tasks[0].Containers) > 0 {
				if res.Tasks[0].Containers[0].LastStatus != nil {
					status = *res.Tasks[0].Containers[0].LastStatus
				}

				if res.Tasks[0].Containers[0].ExitCode != nil {
					exit = int(*res.Tasks[0].Containers[0].ExitCode)
				}
			}
		}

		if exit == 1 {
			return fmt.Errorf("Container exited with non-zero exit code: %d", exit)
		}

		if status == "STOPPED" && exit == 0 {
			return nil
		}

		time.Sleep(delay)
	}

	return fmt.Errorf("timed out waiting for oneshot")
}
