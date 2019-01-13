set -euo pipefail

# TODO: check intermediate results to pin-point potential issue
# kubectl --context cluster1 get secret -l multicluster.admiralty.io/service-account-import.name=cluster2-default-pod-lister -o yaml
# kubectl --context cluster1 get pod -l job-name=multicluster-client -o yaml

POD_NAME_2=$(kubectl --context cluster2 get pod -o jsonpath={.items[0].metadata.name})
kubectl --context cluster1 wait job/multicluster-client --for condition=complete
POD_NAME_1=$(kubectl --context cluster1 logs job/multicluster-client | tail -1)
if [ $POD_NAME_1 == $POD_NAME_2 ]
then
	echo "SUCCESS"
	exit 0
else
	echo "FAILURE"
	exit 1
fi
