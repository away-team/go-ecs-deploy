package ecsdeploy

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/ecs"
)

func (e *ECSDeployer) deployService(taskDef *ecs.RegisterTaskDefinitionInput, service *ecs.CreateServiceInput) error {

	task, err := e.registerTask(taskDef)
	if err != nil {
		return err
	}

	// We must check if the service already exists.
	// Depending on current state we will either Create a new service, or Update an existing service.
	// Currently there is no "upsert" like functionality as part of the ECS SDK.
	svcs, err := e.client.DescribeServices(&ecs.DescribeServicesInput{
		Cluster: service.Cluster,
		Services: []*string{
			service.ServiceName,
		},
	})
	if err != nil {
		return err
	}

	action := "create"
	var s *ecs.Service

	// AWS enforces unique service names per cluster so this is unlikely,
	// but let's guard against multiple services in the response just in case.
	if len(svcs.Services) > 1 {
		return fmt.Errorf("More than one running service matches the provided name: %v", service.ServiceName)
	}

	if len(svcs.Services) == 1 && isServiceActive(svcs.Services[0]) {
		action = "update"
	}

	switch action {
	case "create":
		log.Printf("Creating service...")
		s, err = e.createService(task, service)
		if err != nil {
			return err
		}
	case "update":
		log.Printf("Updating service...")

		if *service.DesiredCount != *svcs.Services[0].DesiredCount {
			log.Printf("Setting desired count to match running service: %d -> %d", *service.DesiredCount, *svcs.Services[0].DesiredCount)
			service.DesiredCount = svcs.Services[0].DesiredCount
		}

		s, err = e.updateService(task, service)

		if err != nil {
			return err
		}
	}

	log.Printf("waiting for stable service state...")
	return e.waitForServiceHealthy(s)
}

func isServiceActive(s *ecs.Service) bool {
	if s.Status != nil {
		if *s.Status == "ACTIVE" {
			return true
		}
	}
	return false
}

// waits for status == ACTIVE && RunningCount == DesiredCount, for 4 consecutive intervals
func (e *ECSDeployer) waitForServiceHealthy(service *ecs.Service) error {
	maxAttempts := 20
	delay := 6 * time.Second
	minHealthyCount := 6
	healthyCount := 0

	expectedDeployments := 1
	expectedRunning := *service.DesiredCount

	for i := 0; i <= maxAttempts; i++ {

		res, err := e.client.DescribeServices(&ecs.DescribeServicesInput{
			Cluster: makeStrPtr(e.cluster),
			Services: []*string{
				service.ServiceName,
			},
		})
		if err != nil {
			return err
		}

		var running int64
		var deployments int

		// assume 1 service with 1 container
		if len(res.Services) > 0 {
			running = *res.Services[0].RunningCount
			deployments = len(res.Services[0].Deployments)
			// fmt.Printf("% #v", pretty.Formatter(res.Services[0]))
		}

		if deployments == expectedDeployments && running == expectedRunning {
			healthyCount++

			// log.Printf("service healthy loop: %d/%d", healthyCount, minHealthyCount)
			if healthyCount == minHealthyCount {
				return nil
			}

		} else {
			healthyCount = 0
		}

		log.Printf("deployments: %d/%d, running: %d/%d, healthyCount: %d/%d", deployments, expectedDeployments, running, expectedRunning, healthyCount, minHealthyCount)

		time.Sleep(delay)
	}

	return fmt.Errorf("timed out waiting for service")
}
