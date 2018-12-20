package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/promoboxx/go-ecs-deploy/src/ecsdeploy"
)

// ./deploy -config <file> -template <file> -type ( oneshot | service ) [ -count int -wait int]

const (
	configFilename            = "config.json"
	environmentConfigFilename = "environment.json"
	ServiceTypeService        = "service"
	ServiceTypeOneshot        = "oneshot"
	envSHA                    = "SHA"
)

type Config struct {
	Service ecs.CreateServiceInput
	Task    ecs.RegisterTaskDefinitionInput
}

func main() {
	service := flag.String("service", "", "The path to the service config file to use")
	task := flag.String("task", "", "The path to the task config file to use")
	serviceType := flag.String("type", "", "The type of service to deploy {service | oneshot}")
	count := flag.Int("count", 40, "The number of iterations to wait for healthy status")
	wait := flag.Int("wait", 5, "The number of seconds to wait between health checks")
	flag.Parse()

	if *service == "" {
		log.Println("-service cannot be blank")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *task == "" {
		log.Println("-task cannot be blank")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *serviceType == "" {
		log.Println("type cannot be blank")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *serviceType != ServiceTypeService && *serviceType != ServiceTypeOneshot {
		log.Printf("Invalid service type: (%s)", *serviceType)
		flag.PrintDefaults()
		os.Exit(1)
	}

	// clean up the passed in config dir path
	serviceConfigPath := filepath.Clean(*service)
	serviceConfgBytes, err := ioutil.ReadFile(serviceConfigPath)
	if err != nil {
		log.Printf("Failed reading service config file: %v", err)
		os.Exit(1)
	}

	// Parse service config
	serviceConf := ecs.CreateServiceInput{}
	err = json.Unmarshal(serviceConfgBytes, &serviceConf)
	if err != nil {
		log.Printf("Failed parsing service config file: %v", err)
		os.Exit(1)
	}

	taskConfigPath := filepath.Clean(*task)
	taskConfgBytes, err := ioutil.ReadFile(taskConfigPath)
	if err != nil {
		log.Printf("Failed reading task config file: %v", err)
		os.Exit(1)
	}
	taskConf := ecs.RegisterTaskDefinitionInput{}
	err = json.Unmarshal(taskConfgBytes, &taskConf)
	if err != nil {
		log.Printf("Failed parsing task config file: %v", err)
		os.Exit(1)
	}

	// Deploy
	// Create the deployer
	deployer := ecsdeploy.NewECSDeployer(*serviceConf.Cluster)
	var deployErr error

	if *serviceType == ServiceTypeService {
		log.Printf("Deploying service...")
		deployErr = deployer.DeployService(&taskConf, &serviceConf, *count, *wait)
	} else {
		log.Printf("Deploying oneshot...")
		deployErr = deployer.DeployOneshot(&taskConf, *count, *wait)
	}

	if deployErr != nil {
		log.Fatalf("Deployment failed: %v", deployErr)
	}

	log.Printf("Deploy success")
	os.Exit(0)
}
