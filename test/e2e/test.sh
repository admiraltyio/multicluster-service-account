set -euo pipefail

kustomize build test/e2e/cluster2/ | kubectl --context cluster2 apply -f -
kubectl --context cluster1 label ns default multicluster-service-account=enabled --overwrite
kubectl config use-context cluster1 && skaffold run -f test/e2e/cluster1/skaffold.yaml

POD_NAME_2=$(kubectl --context cluster2 get pod -o jsonpath={.items[0].metadata.name})
echo "waiting for test job to complete..."
kubectl --context cluster1 wait job/multicluster-client --for condition=complete
POD_NAME_1=$(kubectl --context cluster1 logs job/multicluster-client | tail -1)
if [ "$POD_NAME_1" == "$POD_NAME_2" ]
then
	echo "SUCCESS"
	exit 0
else
	echo "FAILURE"
	exit 1
fi

kubectl config use-context cluster1 && skaffold delete -f test/e2e/cluster1/skaffold.yaml
kubectl --context cluster1 label ns default multicluster-service-account-
kustomize build test/e2e/cluster2/ | kubectl --context cluster2 delete -f -
