package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/away-team/go-ecs-deploy/src/ecsdeploy"
	"github.com/aws/aws-sdk-go/aws"

	"github.com/alecthomas/template"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/mitchellh/mapstructure"
)

// ./deploy -config <file> -template <file> -type ( oneshot | service ) [ -count int -wait int]

const (
	configFilename            = "config.json"
	environmentConfigFilename = "environment.json"
	ServiceTypeService        = "service"
	ServiceTypeOneshot        = "oneshot"
)

type EnvironmentConfig struct {
	TaskFamily          string
	ServiceName         string
	Cluster             string
	SchedulerIAMRoleArn string
	TaskIAMRoleArn      string
	TargetGroupArn      string
	InitialCount        int64
	Image               string
	ImageTag            string
	MemoryReservation   int64
	CPUReservation      int64
	AWSLogsGroupName    string
	AWSLogsRegion       string
	Environment         []map[string]string
}

type Config struct {
	Service ecs.CreateServiceInput
	Task    ecs.RegisterTaskDefinitionInput
}

func main() {
	config := flag.String("config", "", "The path to the config file to use")
	tpl := flag.String("template", "", "The path to the template to use")
	serviceType := flag.String("type", "", "The type of service to deploy {service | oneshot}")
	count := flag.Int("count", 40, "The number of iterations to wait for healthy status")
	wait := flag.Int("wait", 5, "The number of seconds to wait between health checks")

	debug := flag.Bool("debug", false, "Print the the rendered template and exit without deploying")
	flag.Parse()

	if *config == "" {
		log.Println("-config cannot be blank")
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *tpl == "" {
		log.Println("-template cannot be blank")
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

	var envConfig EnvironmentConfig

	// load environment vars into a map and overlay onto envConfig
	environment := getEnv()
	err := mapstructure.Decode(environment, &envConfig)
	if err != nil {
		log.Printf("Error decoding environment: %v", err)
		os.Exit(1)
	}

	// clean up the passed in config dir path
	configPath := filepath.Clean(*config)
	confgBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("Failed reading config file: %v", err)
		os.Exit(1)
	}

	// Overlay config file over envConfig populated by env vars
	err = json.Unmarshal(confgBytes, &envConfig)
	if err != nil {
		log.Printf("Failed parsing config file: %v", err)
		os.Exit(1)
	}

	// clean up the passed in template dir path
	tplPath := filepath.Clean(*tpl)
	f, err := ioutil.ReadFile(tplPath)
	if err != nil {
		log.Printf("Error loading template: %v", err)
		os.Exit(1)
	}

	c, err := runTemplate(f, envConfig)
	if err != nil {
		log.Printf("Failed parsing template: %v", err)
		os.Exit(1)
	}

	var conf Config
	err = json.Unmarshal(c, &conf)
	if err != nil {
		log.Printf("Failed parsing template result to config: %v", err)
		os.Exit(1)
	}

	// Convert the config environment map to the proper type for an ECS ContainerDefinition.
	// TODO: figure out how to actually render a go map into a json map correctly, and handle multi container taks.
	env := make([]*ecs.KeyValuePair, 0)
	for _, m := range envConfig.Environment {
		env = append(env, &ecs.KeyValuePair{Name: aws.String(m["name"]), Value: aws.String(m["value"])})
	}
	conf.Task.ContainerDefinitions[0].Environment = env

	if *debug {
		fmt.Println(string(c))
		os.Exit(0)
	}

	// Deploy

	// Create the deployer
	deployer := ecsdeploy.NewECSDeployer(envConfig.Cluster)
	var deployErr error

	if *serviceType == ServiceTypeService {
		log.Printf("Deploying service...")
		deployErr = deployer.DeployService(&conf.Task, &conf.Service, *count, *wait)
	} else {
		log.Printf("Deploying oneshot...")
		deployErr = deployer.DeployOneshot(&conf.Task, *count, *wait)
	}

	if deployErr != nil {
		log.Fatalf("Deployment failed: %v", deployErr)
	}

	log.Printf("Deploy success")
	os.Exit(0)
}

func getEnv() map[string]string {
	env := make(map[string]string)
	for _, v := range os.Environ() {
		pair := strings.SplitN(v, "=", 2)
		env[pair[0]] = pair[1]
	}
	return env
}

func runTemplate(tpl []byte, data interface{}) ([]byte, error) {

	t, err := template.New("config").Parse(string(tpl))
	if err != nil {
		return nil, fmt.Errorf("Failed parsing template: %v", err)
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
