#!/bin/bash

function init() {
  $HOME/Projects/go/bin/openshift ex new-project test
  $HOME/Projects/go/bin/openshift cli create -n test -f ~/Projects/origin/templates/hello-hooks-deployment.json
}

function deploy() {
  curl -k --cacert openshift.local.certificates/admin/cert.crt --cert openshift.local.certificates/admin/cert.crt --key openshift.local.certificates/admin/key.key https://localhost:8443/osapi/v1beta1/generateDeploymentConfigs/hello?namespace=test | $HOME/Projects/go/bin/openshift cli update -n test deploymentConfigs hello -f -
}

function update() {
  $HOME/Projects/go/bin/openshift cli get -n test deploymentConfigs hello -o json | sed s/Abort/Ignore/ | $HOME/Projects/go/bin/openshift cli update -n test deploymentConfigs hello -f -
  $HOME/Projects/go/bin/openshift cli rollback -n test hello-1
}

function logs() {
  docker ps -a | grep deployer | head -n 1 | awk '{ print $1 }' | xargs docker logs -f
}

if [ "$1" == "init" ]; then init; fi
if [ "$1" == "deploy" ]; then deploy; fi
if [ "$1" == "update" ]; then update; fi
if [ "$1" == "logs" ]; then logs; fi
