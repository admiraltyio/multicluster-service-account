set -euo pipefail

kustomize build test/e2e/cluster1/ | kubectl --context cluster1 delete -f -
kustomize build test/e2e/cluster2/ | kubectl --context cluster2 delete -f -
kubectl config use-context cluster1 && skaffold delete -f test/e2e/cluster1/skaffold.yaml
