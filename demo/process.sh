#!/bin/bash
config=$1

if ! [ -f $config ]; then
  echo "config not found: $config"
  exit 1
fi

version=$(openshift cli get -n demo deploymentConfig demo-deployment -o template -t "{{.metadata.resourceVersion}}")

sed s/RESOURCE_VERSION/$version/ $config
