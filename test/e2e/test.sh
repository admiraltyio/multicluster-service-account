set -euo pipefail

VERSION="$1"

source test/e2e/aliases.sh

k2 apply -f test/e2e/cluster2
k1 label ns default multicluster-service-account=enabled --overwrite
cat test/e2e/cluster1/*.yaml | sed "s/MY_VERSION/$VERSION/g" | k1 apply -f -

POD_NAME_2=$(k2 get pod -l app=nginx -o jsonpath='{.items[0].metadata.name}')
echo "waiting for test job to complete..."
k1 wait job/multicluster-client --for condition=complete --timeout=60s
POD_NAME_1=$(k1 logs job/multicluster-client | tail -1)
if [ "$POD_NAME_1" == "$POD_NAME_2" ]; then
  echo "SUCCESS"
else
  echo "FAILURE"
  exit 1
fi

cat test/e2e/cluster1/*.yaml | sed "s/MY_VERSION/$VERSION/g" | k1 delete -f -
k1 label ns default multicluster-service-account-
k2 delete -f test/e2e/cluster2
