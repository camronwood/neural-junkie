package maps

import (
	"testing"
	"time"
)

func TestLocationStorePublishGetClear(t *testing.T) {
	s := NewLocationStore()
	view := s.Publish(DeviceSnapshot{Lat: 30.27, Lon: -97.74, Shared: true, DisplayName: "Austin, TX"})
	if !view.Shared || view.Lat != 30.27 {
		t.Fatalf("unexpected view: %+v", view)
	}
	got, ok := s.Get()
	if !ok || got.DisplayName != "Austin, TX" {
		t.Fatalf("get: ok=%v view=%+v", ok, got)
	}
	s.Clear()
	if _, ok := s.Get(); ok {
		t.Fatal("expected empty after clear")
	}
}

func TestFreshSharedTTL(t *testing.T) {
	s := NewLocationStore()
	s.Publish(DeviceSnapshot{Lat: 1, Lon: 2, Shared: false})
	if _, ok := s.FreshShared(); ok {
		t.Fatal("unshared snapshot must not be fresh-shared")
	}
	s.Publish(DeviceSnapshot{Lat: 1, Lon: 2, Shared: true, CapturedAt: time.Now().Add(-3 * time.Minute)})
	if _, ok := s.FreshShared(); ok {
		t.Fatal("stale shared snapshot must not be fresh")
	}
	s.Publish(DeviceSnapshot{Lat: 1, Lon: 2, Shared: true})
	if _, ok := s.FreshShared(); !ok {
		t.Fatal("fresh shared snapshot should be available")
	}
}

func TestLocateRequestFulfillAndReject(t *testing.T) {
	s := NewLocationStore()
	req := s.RequestLocate("a1", "Assistant", "general")
	if len(s.ListPending()) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(s.ListPending()))
	}
	done := make(chan *LocateRequest, 1)
	go func() {
		out, err := s.WaitLocate(req.ID, time.Second)
		if err != nil {
			t.Errorf("wait: %v", err)
		}
		done <- out
	}()
	time.Sleep(20 * time.Millisecond)
	if _, err := s.Fulfill(req.ID, DeviceSnapshot{Lat: 41.88, Lon: -87.62}); err != nil {
		t.Fatal(err)
	}
	got := <-done
	if got == nil || got.Status != "fulfilled" || got.Snapshot == nil || got.Snapshot.Lat != 41.88 {
		t.Fatalf("fulfilled: %+v", got)
	}

	req2 := s.RequestLocate("a1", "Assistant", "general")
	if _, err := s.Reject(req2.ID, "nope"); err != nil {
		t.Fatal(err)
	}
	out, err := s.WaitLocate(req2.ID, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "rejected" {
		t.Fatalf("status %s", out.Status)
	}
}
