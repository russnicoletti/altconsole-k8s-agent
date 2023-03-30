#!/bin/sh
ALTC_CONFIG_MAP_NAME="altc-agent"

minikube kubectl -- get configmap $ALTC_CONFIG_MAP_NAME > /dev/null 2>&1
if [ "$?" -eq "0" ]; then
  echo "ConfigMap '${ALTC_CONFIG_MAP_NAME}' already exists"
else
    # The grep found only 0-9, so it's an integer.
    # We can safely do a test on it.
  echo "ConfigMap '${ALTC_CONFIG_MAP_NAME}' does not exist. Creating ..."
  minikube kubectl -- create configmap $ALTC_CONFIG_MAP_NAME --from-literal=CLUSTER_NAME=`kubectl config view --minify -o jsonpath='{.clusters[].name}'`
fi

echo "${ALTC_CONFIG_MAP_NAME} ConfigMap contents:"
echo ""
minikube kubectl -- describe configmap $ALTC_CONFIG_MAP_NAME | grep -A 2 "CLUSTER_NAME"

echo ""
echo "installing agent..."
minikube kubectl -- apply -f helm/templates/altc-agent.yaml
