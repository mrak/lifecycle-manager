package service

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elb/elbiface"

	"github.com/keikoproj/lifecycle-manager/pkg/log"
)

func waitForDeregisterInstance(event *LifecycleEvent, elbClient elbiface.ELBAPI, elbName, instanceID string) error {
	var (
		found bool
	)

	input := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(elbName),
	}

	for i := 0; i < WaiterMaxAttempts; i++ {

		if event.eventCompleted {
			return errors.New("event finished execution during deregistration wait")
		}

		found = false
		instances, err := elbClient.DescribeInstanceHealth(input)
		if err != nil {
			return err
		}
		for _, state := range instances.InstanceStates {
			if aws.StringValue(state.InstanceId) == instanceID {
				found = true
				if aws.StringValue(state.State) == "OutOfService" {
					return nil
				}
				break
			}
		}
		if !found {
			log.Debugf("instance %v not found in elb %v", instanceID, elbName)
			return nil
		}
		log.Debugf("target %v is still deregistering from %v", instanceID, elbName)
		time.Sleep(time.Second * time.Duration(WaiterDelayIntervalSeconds))
	}

	err := errors.New("wait for target deregister timed out")
	return err
}

func findInstanceInClassicBalancer(elbClient elbiface.ELBAPI, elbName, instanceID string) (bool, error) {
	input := &elb.DescribeInstanceHealthInput{
		LoadBalancerName: aws.String(elbName),
	}

	instance, err := elbClient.DescribeInstanceHealth(input)
	if err != nil {
		return false, err
	}
	for _, state := range instance.InstanceStates {
		if aws.StringValue(state.InstanceId) == instanceID {
			return true, nil
		}
	}
	return false, nil
}

func deregisterInstance(elbClient elbiface.ELBAPI, elbName, instanceID string) error {
	input := &elb.DeregisterInstancesFromLoadBalancerInput{
		LoadBalancerName: aws.String(elbName),
		Instances: []*elb.Instance{
			{
				InstanceId: aws.String(instanceID),
			},
		},
	}

	_, err := elbClient.DeregisterInstancesFromLoadBalancer(input)
	if err != nil {
		return err
	}
	return nil
}
