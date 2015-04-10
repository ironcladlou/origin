#!/bin/bash

docker stop $(docker ps -aq) 2>/dev/null
docker rm $(docker ps -aq) 2>/dev/null
rm -rf openshift.local.{volumes,etcd} && sudo /home/dmace/Projects/go/bin/openshift start
