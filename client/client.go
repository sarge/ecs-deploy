package client

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// Client collects the ecs service and logger
type Client struct {
	svc          *ecs.ECS
	logger       *log.Logger
	pollInterval time.Duration
}

// New creates a new client
func New(region *string, profile *string, roleArn *string, logger *log.Logger) *Client {

	var sess *session.Session

	config := &aws.Config{
		Region: region,
	}

	if *profile != "" {
		sess = session.Must(session.NewSessionWithOptions(
			session.Options{
				SharedConfigState: session.SharedConfigEnable,
				Profile:           *profile,
			},
		))
		logger.Printf("adding profile: %s", *profile)

	} else {
		sess = session.New(config)
	}

	if *roleArn != "" {
		creds := stscreds.NewCredentials(sess, *roleArn)

		config = &aws.Config{
			Credentials: creds,
			Region:      region,
		}
		logger.Printf("assuming role: %s", *roleArn)
	}

	svc := ecs.New(sess, config)

	return &Client{
		svc:          svc,
		pollInterval: time.Second * 5,
		logger:       logger,
	}
}

// RegisterTaskDefinition updates the existing task definition's image.
func (c *Client) RegisterTaskDefinition(task, image, tag *string, env *map[string]string) (string, error) {
	taskDef, err := c.GetTaskDefinition(task)
	if err != nil {
		return "", err
	}

	defs := taskDef.ContainerDefinitions
	for _, d := range defs {

		// update the image definition
		if *image != "" && strings.HasPrefix(*d.Image, *image) {
			c.logger.Printf("Updating image to : %s", *image)
			i := fmt.Sprintf("%s:%s", *image, *tag)
			d.Image = &i
		}

		c.merge(d, env)
	}

	input := &ecs.RegisterTaskDefinitionInput{
		Family:               task,
		TaskRoleArn:          taskDef.TaskRoleArn,
		NetworkMode:          taskDef.NetworkMode,
		ContainerDefinitions: defs,
		Volumes:              taskDef.Volumes,
		PlacementConstraints: taskDef.PlacementConstraints,
	}
	resp, err := c.svc.RegisterTaskDefinition(input)
	if err != nil {
		return "", err
	}

	c.logger.Printf("Registered task %s", resp.TaskDefinition)

	return *resp.TaskDefinition.TaskDefinitionArn, nil
}

func (c *Client) merge(containerDefinition *ecs.ContainerDefinition, env *map[string]string) {

	// update the environment variables
	for k, v := range *env {
		found := false

		for _, mv := range containerDefinition.Environment {
			if *mv.Name == k {
				c.logger.Printf("Updating env: %s", k)
				found = true
				mv.SetValue(v)
			}
		}

		if !found {
			c.logger.Printf("Adding env: %s", k)

			kk := k
			vv := v

			containerDefinition.Environment = append(containerDefinition.Environment, &ecs.KeyValuePair{
				Name:  &kk,
				Value: &vv,
			})
		}
	}
}

// UpdateService updates the service to use the new task definition.
func (c *Client) UpdateService(cluster, service *string, count *int64, arn *string) error {
	input := &ecs.UpdateServiceInput{
		Cluster: cluster,
		Service: service,
	}
	if *count != -1 {
		input.DesiredCount = count
	}
	if arn != nil {
		input.TaskDefinition = arn
	}
	_, err := c.svc.UpdateService(input)
	return err
}

// Wait waits for the service to finish being updated.
func (c *Client) Wait(cluster, service, arn *string) error {
	t := time.NewTicker(c.pollInterval)
	for {
		select {
		case <-t.C:
			s, err := c.GetDeployment(cluster, service, arn)
			if err != nil {
				return err
			}
			c.logger.Printf("[info] --> desired: %d, pending: %d, running: %d", *s.DesiredCount, *s.PendingCount, *s.RunningCount)
			if *s.RunningCount == *s.DesiredCount {
				return nil
			}
		}
	}
}

// GetDeployment gets the deployment for the arn.
func (c *Client) GetDeployment(cluster, service, arn *string) (*ecs.Deployment, error) {
	input := &ecs.DescribeServicesInput{
		Cluster:  cluster,
		Services: []*string{service},
	}
	output, err := c.svc.DescribeServices(input)
	if err != nil {
		return nil, err
	}
	ds := output.Services[0].Deployments
	for _, d := range ds {
		if *d.TaskDefinition == *arn {
			return d, nil
		}
	}
	return nil, nil
}

// GetTaskDefinition gets the latest revision for the given task definition
func (c *Client) GetTaskDefinition(task *string) (*ecs.TaskDefinition, error) {
	output, err := c.svc.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: task,
	})
	if err != nil {
		return nil, err
	}
	return output.TaskDefinition, nil
}
