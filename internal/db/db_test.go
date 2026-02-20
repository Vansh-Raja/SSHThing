package db_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/Vansh-Raja/SSHThing/internal/db"
)

func TestDatabaseOperations(t *testing.T) {
	// Create a unique temp directory for this test
	tempDir, err := os.MkdirTemp("", "ssh-manager-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Force DB into our temp directory (cross-platform)
	originalDataDir := os.Getenv("SSHTHING_DATA_DIR")
	os.Setenv("SSHTHING_DATA_DIR", tempDir)
	defer os.Setenv("SSHTHING_DATA_DIR", originalDataDir)

	// Test 1: First run - setup should work
	t.Run("FirstRunSetup", func(t *testing.T) {
		exists, err := db.Exists()
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if exists {
			t.Fatal("DB should not exist yet")
		}

		// Create DB with password
		store, err := db.Init("testpassword123")
		if err != nil {
			t.Fatalf("Init() failed: %v", err)
		}
		defer store.Close()

		fmt.Println("✓ First run setup works")
	})

	// Test 2: DB should now exist
	t.Run("DBExists", func(t *testing.T) {
		exists, err := db.Exists()
		if err != nil {
			t.Fatalf("Exists() failed: %v", err)
		}
		if !exists {
			t.Fatal("DB should exist now")
		}
		fmt.Println("✓ DB exists check works")
	})

	// Test 3: Wrong password should fail
	t.Run("WrongPassword", func(t *testing.T) {
		_, err := db.Init("wrongpassword")
		if err == nil {
			t.Fatal("Wrong password should fail")
		}
		fmt.Printf("✓ Wrong password correctly rejected: %v\n", err)
	})

	// Test 4: Correct password should work
	t.Run("CorrectPassword", func(t *testing.T) {
		store, err := db.Init("testpassword123")
		if err != nil {
			t.Fatalf("Correct password should work: %v", err)
		}
		defer store.Close()
		fmt.Println("✓ Correct password works")
	})

	// Test 5: Add and retrieve host
	t.Run("AddAndGetHost", func(t *testing.T) {
		store, err := db.Init("testpassword123")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		defer store.Close()

		// Add a host
		host := &db.HostModel{
			Label:    "test",
			Hostname: "test.example.com",
			Username: "testuser",
			Port:     22,
			KeyType:  "password",
		}

		err = store.CreateHost(host, "")
		if err != nil {
			t.Fatalf("CreateHost failed: %v", err)
		}
		fmt.Println("✓ Host created")

		// Get hosts
		hosts, err := store.GetHosts()
		if err != nil {
			t.Fatalf("GetHosts failed: %v", err)
		}

		if len(hosts) != 1 {
			t.Fatalf("Expected 1 host, got %d", len(hosts))
		}

		if hosts[0].Hostname != "test.example.com" {
			t.Fatalf("Hostname mismatch: %s", hosts[0].Hostname)
		}
		if hosts[0].KeyData != "" {
			t.Fatalf("Expected empty key_data for password auth, got: %q", hosts[0].KeyData)
		}

		fmt.Printf("✓ GetHosts works: %+v\n", hosts[0])
	})

	// Test 6: Groups (create, assign, rename, delete)
	t.Run("Groups", func(t *testing.T) {
		store, err := db.Init("testpassword123")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		defer store.Close()

		// Create group
		if err := store.UpsertGroup("Work"); err != nil {
			t.Fatalf("UpsertGroup failed: %v", err)
		}

		// Add a host in that group
		host := &db.HostModel{
			Label:     "prod",
			GroupName: "Work",
			Hostname:  "prod.example.com",
			Username:  "ubuntu",
			Port:      22,
			KeyType:   "password",
		}
		if err := store.CreateHost(host, ""); err != nil {
			t.Fatalf("CreateHost failed: %v", err)
		}

		hosts, err := store.GetHosts()
		if err != nil {
			t.Fatalf("GetHosts failed: %v", err)
		}
		found := false
		for _, h := range hosts {
			if h.Hostname == "prod.example.com" {
				found = true
				if h.GroupName != "Work" {
					t.Fatalf("expected group 'Work', got %q", h.GroupName)
				}
			}
		}
		if !found {
			t.Fatalf("expected to find inserted host")
		}

		// Rename group and ensure host follows
		if err := store.RenameGroup("Work", "Work2"); err != nil {
			t.Fatalf("RenameGroup failed: %v", err)
		}
		hosts, err = store.GetHosts()
		if err != nil {
			t.Fatalf("GetHosts failed: %v", err)
		}
		for _, h := range hosts {
			if h.Hostname == "prod.example.com" && h.GroupName != "Work2" {
				t.Fatalf("expected group 'Work2' after rename, got %q", h.GroupName)
			}
		}

		// Delete group and ensure host is ungrouped
		if err := store.DeleteGroup("Work2"); err != nil {
			t.Fatalf("DeleteGroup failed: %v", err)
		}
		hosts, err = store.GetHosts()
		if err != nil {
			t.Fatalf("GetHosts failed: %v", err)
		}
		for _, h := range hosts {
			if h.Hostname == "prod.example.com" && h.GroupName != "" {
				t.Fatalf("expected host to be ungrouped, got %q", h.GroupName)
			}
		}

		groups, err := store.GetGroups()
		if err != nil {
			t.Fatalf("GetGroups failed: %v", err)
		}
		for _, g := range groups {
			if g == "Work2" {
				t.Fatalf("expected deleted group to be hidden")
			}
		}
	})

	fmt.Println("\n✓ All tests passed!")
}
