package models

import (
	"sync"
	"testing"

	"github.com/gophish/gophish/config"
	"gorm.io/gorm"
)

// TestReadDB_FallbackToWriter verifies that readDB() returns the writer
// connection (db) when no dedicated reader is configured. This is the
// expected behavior for MySQL and :memory: SQLite databases.
func TestReadDB_FallbackToWriter(t *testing.T) {
	oldReader := readerDB
	readerDB = nil
	defer func() { readerDB = oldReader }()

	if got := readDB(); got != db {
		t.Fatal("readDB() should return db when readerDB is nil")
	}
}

// TestReadDB_ReturnsReader verifies that readDB() returns the dedicated
// reader connection when one has been configured (on-disk SQLite).
func TestReadDB_ReturnsReader(t *testing.T) {
	oldReader := readerDB
	defer func() { readerDB = oldReader }()

	fakeReader := &gorm.DB{}
	readerDB = fakeReader

	if got := readDB(); got != fakeReader {
		t.Fatal("readDB() should return readerDB when it is non-nil")
	}
}

// TestConcurrentReads verifies that multiple goroutines can issue read
// queries through readDB() concurrently without panics or data races.
// Uses the real :memory: test database (db set up by SetUpSuite).
func TestConcurrentReads(t *testing.T) {
	// Ensure db is initialised — SetUpSuite may not have run yet for
	// testing.T-based tests.
	if db == nil {
		if err := Setup(&config.Config{
			DBName:         "sqlite3",
			DBPath:         ":memory:",
			MigrationsPath: "../db/db_sqlite3/migrations/",
		}); err != nil {
			t.Fatalf("Setup failed: %v", err)
		}
	}

	const goroutines = 10
	const iterations = 50

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				var count int64
				if err := readDB().Model(&User{}).Count(&count).Error; err != nil {
					errCh <- err
					return
				}
				var cs []Campaign
				if err := readDB().Find(&cs).Error; err != nil {
					errCh <- err
					return
				}
			}
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent read failed: %v", err)
	}
}

// TestCloseDB_NilSafety verifies that CloseDB() does not panic when
// both db and readerDB are nil (e.g. called before Setup).
func TestCloseDB_NilSafety(t *testing.T) {
	oldDB := db
	oldReader := readerDB
	oldConf := conf
	db = nil
	readerDB = nil
	conf = nil
	defer func() {
		db = oldDB
		readerDB = oldReader
		conf = oldConf
	}()

	// Should not panic.
	CloseDB()
}

// TestIsOnDiskSQLite verifies the helper correctly identifies database modes.
func TestIsOnDiskSQLite(t *testing.T) {
	oldConf := conf
	defer func() { conf = oldConf }()

	tests := []struct {
		name string
		conf *config.Config
		want bool
	}{
		{"nil config", nil, false},
		{"mysql", &config.Config{DBName: "mysql", DBPath: "user:pass@tcp(localhost)/db"}, false},
		{"memory", &config.Config{DBName: "sqlite3", DBPath: ":memory:"}, false},
		{"on-disk", &config.Config{DBName: "sqlite3", DBPath: "/tmp/test.db"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf = tt.conf
			if got := isOnDiskSQLite(); got != tt.want {
				t.Errorf("isOnDiskSQLite() = %v, want %v", got, tt.want)
			}
		})
	}
}
