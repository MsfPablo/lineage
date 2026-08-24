//go:build windows

package materialize

import (
	"os"
	"testing"
)

// assertNoLooseWriteBits is a no-op on Windows: chmod there only toggles
// the read-only attribute, with no group/other distinction to strip, so the
// 0o755 cap this test exercises on POSIX has nothing observable to assert
// through os.FileMode here.
func assertNoLooseWriteBits(t *testing.T, mode os.FileMode) {
	t.Helper()
}
