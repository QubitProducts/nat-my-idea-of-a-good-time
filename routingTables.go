package main

import (
	"flag"

	"github.com/golang/glog"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	subnetId              string
	primaryRouteTableId   string
	secondaryRouteTableId string

	awsRegion string
)

func init() {
	flag.StringVar(&subnetId, "subnet", "", "Subnet ID")
	flag.StringVar(&primaryRouteTableId, "primary", "", "Primary route table id")
	flag.StringVar(&secondaryRouteTableId, "secondary", "", "Secondary route table id")
	flag.StringVar(&awsRegion, "region", "eu-west-1", "AWS region that the subnet and route tables are in")
}

func makeRouteTableFailoverAction() Action {
	c := ec2.New(session.New(&aws.Config{Region: aws.String(awsRegion)}))

	validateSubnetId(c, subnetId)
	validateRouteTableId(c, primaryRouteTableId, "primary")
	validateRouteTableId(c, secondaryRouteTableId, "secondary")

	return makeAction(func(err error) error {
		return failoverRouteTable(c, err)
	})
}

func failoverRouteTable(c *ec2.EC2, _ error) error {
	glog.Infof("Moving route table over to %v", secondaryRouteTableId)

	req := &ec2.AssociateRouteTableInput{
		DryRun:       &dryRun,
		RouteTableId: &secondaryRouteTableId,
		SubnetId:     &subnetId,
	}

	_, err := c.AssociateRouteTable(req)
	return err
}

func validateRouteTableId(c *ec2.EC2, id, key string) {
	if id == "" {
		glog.Fatalf("No %v route table id given", key)
	}
	req := ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{&id},
	}

	// Don't need to inspect the result, as a missing value will result in err != nil
	_, err := c.DescribeRouteTables(&req)
	if err != nil {
		glog.Fatalf("Failed to find route table: %v", err)
	}
}

func validateSubnetId(c *ec2.EC2, id string) {
	if id == "" {
		glog.Fatalf("No subnet id given")
	}
	req := ec2.DescribeSubnetsInput{
		SubnetIds: []*string{&id},
	}

	_, err := c.DescribeSubnets(&req)
	if err != nil {
		glog.Fatalf("Failed to find route table: %v", err)
	}
}
