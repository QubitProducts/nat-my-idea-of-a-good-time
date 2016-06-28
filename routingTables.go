package main

import (
	"flag"
	"fmt"

	"github.com/golang/glog"
	"github.com/pkg/errors"

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

	associationId, err := findAssociationId(c, primaryRouteTableId, subnetId)
	if err != nil {
		glog.Errorf("Could not find association ID. This could indicate that we have already failed over. Erroring anyway")
		return errors.Wrap(err, "finding primary route table association id for subnet failed")
	}

	disassocReq := &ec2.DisassociateRouteTableInput{
		DryRun: &dryRun,
		AssociationId: &associationId,
	}
	_, err = c.DisassociateRouteTable(disassocReq)
	if err != nil {
		return errors.Wrap(err, "primary route table disassociation failed")
	}

	assocReq := &ec2.AssociateRouteTableInput{
		DryRun:       &dryRun,
		RouteTableId: &secondaryRouteTableId,
		SubnetId:     &subnetId,
	}

	_, err = c.AssociateRouteTable(assocReq)
	if err != nil {
		return errors.Wrap(err, "secondary route table association failed")
	}

	return nil
}

func findAssociationId(c *ec2.EC2, routeTableId, subnetId string) (string, error) {
	req := ec2.DescribeRouteTablesInput{
		RouteTableIds: []*string{&routeTableId},
	}

	res, err := c.DescribeRouteTables(&req)
	if err != nil {
		return "", err
	}

	if len(res.RouteTables) != 1 {
		return "", fmt.Errorf("Could not find route table %v", routeTableId)
	}
	routeTable := res.RouteTables[0]

	for _, assoc := range routeTable.Associations {
		if *assoc.SubnetId == subnetId {
			return *assoc.RouteTableAssociationId, nil
		}
	}

	return "", fmt.Errorf("Could not find associationID for subnet %v and route table %v", subnetId, routeTableId)
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
