set -euo pipefail

kubectl config use-context cluster1 && skaffold delete -f test/e2e/cluster1/skaffold.yaml
