#!/bin/sh
ALTC_CONFIG_MAP_NAME="altc-agent"
ALTC_SECRET_NAME="altc-agent"
ALTC_CONFIG_MAP_TEMPLATE_FILE="deployment_files/altc-agent-configmap.yaml"

kubectl get configmap $ALTC_CONFIG_MAP_NAME > /dev/null 2>&1
if [ "$?" -eq "0" ]; then
  echo "ConfigMap '${ALTC_CONFIG_MAP_NAME}' already exists"
else
  echo "ConfigMap '${ALTC_CONFIG_MAP_NAME}' does not exist. Creating ..."
  CLUSTER_NAME=`kubectl config view --minify -o jsonpath='{.clusters[].name}'`
  sed "s/REPLACE_WITH_CLUSTER_NAME/${CLUSTER_NAME}/g" ${ALTC_CONFIG_MAP_TEMPLATE_FILE} | kubectl apply -f -
fi

echo "${ALTC_CONFIG_MAP_NAME} ConfigMap contents:"
minikube kubectl -- describe configmap $ALTC_CONFIG_MAP_NAME | grep -A 2 "CLUSTER_NAME"

kubectl get secret $ALTC_SECRET_NAME > /dev/null 2>&1
if [ "$?" -eq "0" ]; then
  echo "secret '${ALTC_SECRET_NAME}' already exists"
else
  echo "secret '${ALTC_SECRET_NAME}' does not exist. Creating ..."
  kubectl apply -f deployment_files/altc-agent-secret.yaml
fi

echo ""
echo "installing agent..."
helm list | grep altc-chart > /dev/null 2>&1
if [ "$?" -eq "0" ]; then
  helm upgrade altc-chart altc-helm/altc-agent
else
  helm install altc-chart altc-helm/altc-agent
fi

