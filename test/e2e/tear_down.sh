set -euo pipefail

kubectl config use-context cluster1 && skaffold delete -f test/e2e/install/skaffold.yaml
kubectl --context cluster2 delete namespace multicluster-service-account # TODO? run kubemcsa unstrap when it exists
