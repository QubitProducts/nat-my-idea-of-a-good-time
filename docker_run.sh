#!/bin/bash
set -e
set -u

AZ=`curl -s http://169.254.169.254/latest/meta-data/placement/availability-zone`

. /secrets.env
. /${AZ}.env

env 1>&2

./nat-my-idea-of-a-good-time -logtostderr -name ${AZ}
