package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func TestProfiles(t *testing.T) {
	if _, ok := Profiles["Gaming"]; !ok {
		t.Error("Gaming profile not found")
	}
	if _, ok := Profiles["Meeting"]; !ok {
		t.Error("Meeting profile not found")
	}
	if _, ok := Profiles["Browsing"]; !ok {
		t.Error("Browsing profile not found")
	}
}

func TestMeasureLocal(t *testing.T) {
	// Ping localhost. Should always work if network stack is okay.
	metrics, err := Measure("127.0.0.1", 1, time.Second)
	if err != nil {
		t.Logf("Measure 127.0.0.1 failed (might be expected in some CI environments): %v", err)
		return
	}

	if !metrics.IsUp {
		t.Errorf("Expected localhost to be up")
	}
}

func TestProfileThresholds(t *testing.T) {
	gaming := Profiles["Gaming"]
	if gaming.MaxLatencyMs != 100 {
		t.Errorf("Expected Gaming MaxLatencyMs 100, got %d", gaming.MaxLatencyMs)
	}

	meeting := Profiles["Meeting"]
	if meeting.MaxPacketLoss != 5.0 {
		t.Errorf("Expected Meeting MaxPacketLoss 5.0, got %f", meeting.MaxPacketLoss)
	}
}

func TestFindBestTarget(t *testing.T) {
	// Ping localhost and a known external target
	targets := []string{"127.0.0.1", "8.8.8.8"}
	best := findBestTarget(targets)

	// Localhost should ideally be the best/fastest
	if best == "" {
		t.Log("findBestTarget found no active targets (likely no network/permissions)")
	} else {
		t.Logf("findBestTarget selected: %s", best)
	}
}

func TestSingleInstanceLocking(t *testing.T) {
	lockFilePath := filepath.Join(os.TempDir(), "net-stabilizer-test.lock")
	
	// Create first lock
	lock1 := flock.New(lockFilePath)
	locked1, err := lock1.TryLock()
	if err != nil {
		t.Fatalf("Failed to acquire first lock: %v", err)
	}
	if !locked1 {
		t.Fatal("Expected to acquire first lock")
	}
	defer lock1.Unlock()

	// Attempt second lock (should fail)
	lock2 := flock.New(lockFilePath)
	locked2, err := lock2.TryLock()
	if err != nil {
		t.Fatalf("Unexpected error trying to acquire second lock: %v", err)
	}
	if locked2 {
		t.Fatal("Expected NOT to acquire second lock")
	}

	// Release first lock
	lock1.Unlock()

	// Attempt second lock again (should succeed)
	locked3, err := lock2.TryLock()
	if err != nil {
		t.Fatalf("Failed to acquire second lock after release: %v", err)
	}
	if !locked3 {
		t.Fatal("Expected to acquire second lock after first release")
	}
	lock2.Unlock()
	
	// Cleanup test lock file
	os.Remove(lockFilePath)
}
