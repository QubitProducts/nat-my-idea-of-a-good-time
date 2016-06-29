#!/bin/bash
set -e
set -u
set -x

AZ=`curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone`

. /secrets.env
. /${AZ}.env

./nat-my-idea-of-a-good-time -logtostderr
