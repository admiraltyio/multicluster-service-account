set -euo pipefail

VERSION="$1"

KUBECONFIG=kubeconfig-cluster_2 kubectl apply -f test/e2e/cluster2
KUBECONFIG=kubeconfig-cluster_1 kubectl label ns default multicluster-service-account=enabled --overwrite
cat test/e2e/cluster1/*.yaml | sed "s/MY_VERSION/$VERSION/g" | KUBECONFIG=kubeconfig-cluster_1 kubectl apply -f -

POD_NAME_2=$(KUBECONFIG=kubeconfig-cluster_2 kubectl get pod -o jsonpath='{.items[0].metadata.name}')
echo "waiting for test job to complete..."
KUBECONFIG=kubeconfig-cluster_1 kubectl wait job/multicluster-client --for condition=complete --timeout=60s
POD_NAME_1=$(KUBECONFIG=kubeconfig-cluster_1 kubectl logs job/multicluster-client | tail -1)
if [ "$POD_NAME_1" == "$POD_NAME_2" ]; then
  echo "SUCCESS"
  exit 0
else
  echo "FAILURE"
  exit 1
fi

cat test/e2e/cluster1/*.yaml | sed "s/MY_VERSION/$VERSION/g" | KUBECONFIG=kubeconfig-cluster_1 kubectl delete -f -
KUBECONFIG=kubeconfig-cluster_1 kubectl label ns default multicluster-service-account-
KUBECONFIG=kubeconfig-cluster_2 kubectl delete -f test/e2e/cluster2
