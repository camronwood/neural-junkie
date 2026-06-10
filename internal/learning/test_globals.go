package learning

import "sync"

// testGlobalsMu serializes tests that wire package-level learning stores.
// go test ./... runs packages in parallel; without this, async RecordUse and
// TempDir cleanup can race across packages (flaky CI on Linux).
var testGlobalsMu sync.Mutex

// LockTestGlobals acquires the learning test lock and registers cleanup that
// waits for async RecordUse, clears wired globals, then unlocks.
func LockTestGlobals() func() {
	testGlobalsMu.Lock()
	return func() {
		WaitPendingRecordUse()
		SetGlobalStore(nil)
		SetEmbedStore(nil)
		SetEnabledChecker(nil)
		testGlobalsMu.Unlock()
	}
}
