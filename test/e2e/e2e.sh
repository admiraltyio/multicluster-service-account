set -euo pipefail

VERSION="$1"

test/e2e/setup_clusters.sh
test/e2e/setup.sh "$VERSION"
test/e2e/test.sh "$VERSION"
test/e2e/test_export.sh
test/e2e/tear_down.sh
test/e2e/tear_down_clusters.sh
