#!/bin/bash

$(boot2docker shellinit 2>/dev/null)

echo "Cleaning up containers..."
docker stop $(docker ps -qa) ; docker rm $(docker ps -qa)
rm -rf openshift.local.{volumes,etcd} && openshift start
