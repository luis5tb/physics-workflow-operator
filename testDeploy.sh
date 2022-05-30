#!/bin/sh
MYIP=$(ifconfig eth0 | grep broadcast | awk '{print $2}')
HUB_CTX=kind-hub
MANAGED_CTX=kind-physics
WORKFLOW_NAME=hello-sequence
NAMESPACE=default
OW_NAMESPACE=guest
MANAGED_CLUSTER=cluster1
OW_AUTH=23bc46b1-71f6-4ed5-8c54-816aa4f8c502:123zO3xZCLrMN6v2BKK1dXYFpXlPkccOFqm12CdAsMgRU4VrNZ9lyGVCGuMDGIwP
echo '*** INIT all...'
kubectl delete -f config/samples/mymanifestworkmulti.yaml --context $HUB_CTX
#wsk --apihost http://$MYIP:3233 --auth $OW_AUTH action delete /$OW_NAMESPACE/$WORKFLOW_NAME/$WORKFLOW_NAME
#wsk --apihost http://$MYIP:3233 --auth $OW_AUTH action delete /$OW_NAMESPACE/$WORKFLOW_NAME/hello
#wsk --apihost http://$MYIP:3233 --auth $OW_AUTH action delete /$OW_NAMESPACE/$WORKFLOW_NAME/world
#wsk --apihost http://$MYIP:3233 --auth $OW_AUTH package delete /$OW_NAMESPACE/$WORKFLOW_NAME 
#wsk --apihost http://$MYIP:3233 --auth $OW_AUTH api delete /$WORKFLOW_NAME 
sleep 3
kubectl -n $NAMESPACE get workflow --context $MANAGED_CTX
wsk --apihost http://$MYIP:3233 --auth $OW_AUTH list
wsk --apihost http://$MYIP:3233 --auth $OW_AUTH api list
#exit
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
wsk --apihost http://$MYIP:3233 --auth $OW_AUTH list
wsk --apihost http://$MYIP:3233 --auth $OW_AUTH api list
echo '*** INVOKING Workflow (action sequence) in OW (MANAGED cluster)...'
wsk --apihost http://$MYIP:3233 --auth $OW_AUTH action invoke /$OW_NAMESPACE/$WORKFLOW_NAME/$WORKFLOW_NAME -p payload Hi -r

echo '*** DEPLOYING ManifestWork in the HUB cluster (PHASE II)...'
kubectl apply -f config/samples/mymanifestworkmulti2.yaml --context $HUB_CTX
echo '*** GETTING Workflow in the MANAGED cluster ...'
kubectl -n $NAMESPACE get workflow $WORKFLOW_NAME -o yaml --context $MANAGED_CTX
echo '*** Workflow deployment status...'
sleep 3
kubectl get workflow $WORKFLOW_NAME -o json --context $MANAGED_CTX | jq .status.conditions
echo '*** INVOKING Workflow (action sequence) in OW (MANAGED cluster)...'
wsk --apihost http://$MYIP:3233 --auth $OW_AUTH action invoke /$OW_NAMESPACE/$WORKFLOW_NAME/$WORKFLOW_NAME -p payload Hi -r
