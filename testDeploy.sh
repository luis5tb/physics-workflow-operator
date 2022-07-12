#!/bin/bash
MYIP=$(ifconfig eth0 | grep broadcast | awk '{print $2}')
OW_APIHOST=http://$MYIP:3233
#OW_APIHOST=openwhisk.apps.ocphub.physics-faas.eu
#HUB_CTX=kind-hub
HUB_CTX=physics-atos/api-ocphub-physics-faas-eu:6443/atos
#MANAGED_CTX=kind-physics
MANAGED_CTX=kind-atos-edge
#MANAGED_CTX=physics-atos/api-ocphub-physics-faas-eu:6443/atos
WORKFLOW_NAME=hello-sequence-atos
NAMESPACE=default
#NAMESPACE=physics-atos
OW_NAMESPACE=guest
#OW_NAMESPACE=physics-atos
#MANAGED_CLUSTER=cluster1
MANAGED_CLUSTER=atos-edge
#MANAGED_CLUSTER=okdhub-cluster
OW_AUTH=23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP
echo '*** INIT all...'
kubectl delete -f config/samples/mymanifestworkmulti.yaml --context $HUB_CTX
#wsk --apihost $OW_APIHOST --auth $OW_AUTH action delete /$OW_NAMESPACE/$WORKFLOW_NAME/$WORKFLOW_NAME
#wsk --apihost $OW_APIHOST --auth $OW_AUTH action delete /$OW_NAMESPACE/$WORKFLOW_NAME/hello
#wsk --apihost $OW_APIHOST --auth $OW_AUTH action delete /$OW_NAMESPACE/$WORKFLOW_NAME/world
#wsk --apihost $OW_APIHOST --auth $OW_AUTH package delete /$OW_NAMESPACE/$WORKFLOW_NAME 
#wsk --apihost $OW_APIHOST --auth $OW_AUTH api delete /$WORKFLOW_NAME 
sleep 3
echo "*** GETTING Workflow in the MANAGED cluster $MANAGED_CLUSTER (K8S namespace $NAMESPACE)..."
kubectl -n $NAMESPACE get workflow --context $MANAGED_CTX
echo "*** LISTING actions in OW (MANAGED cluster) with OW user/namespace $OW_NAMESPACE ..."
wsk -i --apihost $OW_APIHOST --auth $OW_AUTH list
#wsk -i --apihost $OW_APIHOST --auth $OW_AUTH api list
#exit
read -n1 -r -p "Press any key to continue..."
echo '*** DEPLOYING ManifestWork in the HUB cluster (PHASE I)...'
kubectl apply -f config/samples/mymanifestworkmulti.yaml --context $HUB_CTX
kubectl -n $MANAGED_CLUSTER get manifestwork --context $HUB_CTX
echo '*** GETTING Workflow in the MANAGED cluster ...'
kubectl -n $NAMESPACE get workflow --context $MANAGED_CTX
kubectl -n $NAMESPACE get workflow $WORKFLOW_NAME -o yaml --context $MANAGED_CTX
echo '*** Workflow deployment status...'
sleep 3
kubectl get workflow $WORKFLOW_NAME -o json --context $MANAGED_CTX | jq .status.conditions
echo '*** LISTING actions in OW (MANAGED cluster)...'
wsk -i --apihost $OW_APIHOST --auth $OW_AUTH list
#wsk -i --apihost $OW_APIHOST --auth $OW_AUTH api list
echo '*** INVOKING Workflow (action sequence) in OW (MANAGED cluster)...'
wsk -i --apihost $OW_APIHOST --auth $OW_AUTH action invoke /$OW_NAMESPACE/$WORKFLOW_NAME/$WORKFLOW_NAME -p payload Hi -r
#exit
read -n1 -r -p "Press any key to continue..."
echo '*** DEPLOYING ManifestWork in the HUB cluster (PHASE II)...'
kubectl apply -f config/samples/mymanifestworkmulti2.yaml --context $HUB_CTX
echo '*** GETTING Workflow in the MANAGED cluster ...'
kubectl -n $NAMESPACE get workflow $WORKFLOW_NAME -o yaml --context $MANAGED_CTX
echo '*** Workflow deployment status...'
sleep 3
kubectl get workflow $WORKFLOW_NAME -o json --context $MANAGED_CTX | jq .status.conditions
echo '*** INVOKING Workflow (action sequence) in OW (MANAGED cluster)...'
wsk -i --apihost $OW_APIHOST --auth $OW_AUTH action invoke /$OW_NAMESPACE/$WORKFLOW_NAME/$WORKFLOW_NAME -p payload Hi -r
