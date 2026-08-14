package models

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"bitbucket.org/liamstask/goose/lib/goose"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/gophish/gophish/auth"
	"github.com/gophish/gophish/config"

	log "github.com/gophish/gophish/logger"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB
var readerDB *gorm.DB // separate reader connection for SQLite read-write splitting
var conf *config.Config

// defaultReaderMaxConns is the default MaxOpenConns for the reader connection
// when the user does not specify db_reader_max_conns in config.json.
// SQLite in WAL mode supports multiple concurrent readers without blocking the
// single writer, so a modest pool size (5) significantly improves read
// throughput compared to the previous MaxOpenConns(1).
const defaultReaderMaxConns = 5

// isOnDiskSQLite reports whether the configured database is a file-based
// SQLite database (as opposed to MySQL or :memory: SQLite). Both the
// startup WAL checkpoint and the reader connection are only relevant for
// on-disk SQLite.
func isOnDiskSQLite() bool {
	return conf != nil && conf.DBName != "mysql" && !strings.HasPrefix(conf.DBPath, ":memory:")
}

// readDB returns the connection used for read-only queries. Falls back to
// the writer connection when no dedicated reader is configured (e.g. MySQL,
// in-memory test databases), so callers don't need to know which mode is
// active.
func readDB() *gorm.DB {
	if readerDB != nil {
		return readerDB
	}
	return db
}

// CloseDB closes both writer and reader database connections gracefully.
// For SQLite, it explicitly triggers a WAL checkpoint (TRUNCATE mode) so
// that all data in the WAL file is flushed to the main .db file.
// The reader connection is closed first so that no open readers remain
// when the checkpoint runs, allowing WAL truncation to succeed.
func CloseDB() {
	if db == nil {
		return
	}

	// 1. Close the reader connection first so it releases the WAL read lock.
	if readerDB != nil {
		rSqlDB, err := readerDB.DB()
		if err != nil {
			log.Errorf("failed to get reader sql.DB: %v", err)
		} else if err := rSqlDB.Close(); err != nil {
			log.Errorf("failed to close reader database: %v", err)
		} else {
			log.Info("Reader database connection closed")
		}
		readerDB = nil
	}

	// 2. Get the underlying sql.DB — needed for both checkpoint and close.
	sqlDB, err := db.DB()
	if err != nil {
		log.Errorf("failed to get underlying sql.DB: %v", err)
		return
	}

	// 3. Explicit WAL checkpoint BEFORE closing the writer connection.
	//    Use the raw *sql.DB.Exec instead of GORM's db.Exec because GORM v2
	//    may wrap PRAGMA statements in ways that prevent proper execution.
	//    TRUNCATE mode resets the WAL file to zero bytes after checkpoint.
	if isOnDiskSQLite() {
		if _, err := sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			log.Warnf("WAL checkpoint failed (non-fatal): %v", err)
		} else {
			log.Info("SQLite WAL checkpoint completed")
		}
	}

	// 4. Close the main writer connection pool.
	if err := sqlDB.Close(); err != nil {
		log.Errorf("failed to close writer database: %v", err)
	} else {
		log.Info("Writer database connection closed")
	}
}

const MaxDatabaseConnectionAttempts int = 10

// DefaultAdminUsername is the default username for the administrative user
const DefaultAdminUsername = "admin"

// InitialAdminPassword is the environment variable that specifies which
// password to use for the initial root login instead of generating one
// randomly
const InitialAdminPassword = "GOPHISH_INITIAL_ADMIN_PASSWORD"

// InitialAdminApiToken is the environment variable that specifies the
// API token to seed the initial root login instead of generating one
// randomly
const InitialAdminApiToken = "GOPHISH_INITIAL_ADMIN_API_TOKEN"

const (
	CampaignInProgress string = "In progress"
	CampaignQueued     string = "Queued"
	CampaignCreated    string = "Created"
	CampaignScheduled  string = "Scheduled"
	CampaignEmailsSent string = "Emails Sent"
	CampaignComplete   string = "Completed"
	EventSent          string = "Email Sent"
	EventSendingError  string = "Error Sending Email"
	EventOpened        string = "Email Opened"
	EventClicked       string = "Clicked Link"
	EventDataSubmit    string = "Submitted Data"
	EventReported      string = "Email Reported"
	EventProxyRequest  string = "Proxied request"
	StatusSuccess      string = "Success"
	StatusQueued       string = "Queued"
	StatusSending      string = "Sending"
	StatusUnknown      string = "Unknown"
	StatusScheduled    string = "Scheduled"
	StatusRetry        string = "Retrying"
	Error              string = "Error"
)

// Flash is used to hold flash information for use in templates.
type Flash struct {
	Type    string
	Message string
}

// Response contains the attributes found in an API response
type Response struct {
	Message string      `json:"message"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

// Copy of auth.GenerateSecureKey to prevent cyclic import with auth library
func generateSecureKey() string {
	k := make([]byte, 32)
	io.ReadFull(rand.Reader, k)
	return fmt.Sprintf("%x", k)
}

// getDBConnectionString returns the connection string used to open the
// database. For on-disk SQLite databases we enable WAL mode, use a normal
// synchronous mode and configure a busy timeout. WAL mode allows readers to
// proceed while a writer holds the database, and the busy timeout prevents
// "database is locked" errors by waiting for the lock to be released instead
// of failing immediately. In-memory databases (used in tests) don't support
// WAL, so they are returned untouched.
func getDBConnectionString(c *config.Config) string {
	if c.DBName == "mysql" {
		return c.DBPath
	}
	if strings.HasPrefix(c.DBPath, ":memory:") {
		return c.DBPath
	}
	// If the path already contains query parameters (e.g. a user configured
	// their own DSN), leave it untouched rather than appending a malformed
	// fragment.
	if strings.Contains(c.DBPath, "?") {
		return c.DBPath
	}
	return fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000", c.DBPath)
}

func chooseDBDriver(name, openStr string) goose.DBDriver {
	d := goose.DBDriver{Name: name, OpenStr: openStr}

	switch name {
	case "mysql":
		d.Import = "github.com/go-sql-driver/mysql"
		d.Dialect = &goose.MySqlDialect{}

	// Default database is sqlite3
	default:
		d.Import = "github.com/mattn/go-sqlite3"
		d.Dialect = &goose.Sqlite3Dialect{}
	}

	return d
}

func createTemporaryPassword(u *User) error {
	var temporaryPassword string
	if envPassword := os.Getenv(InitialAdminPassword); envPassword != "" {
		temporaryPassword = envPassword
	} else {
		// This will result in a 16 character password which could be viewed as an
		// inconvenience, but it should be ok for now.
		temporaryPassword = auth.GenerateSecureKey(auth.MinPasswordLength)
	}
	hash, err := auth.GeneratePasswordHash(temporaryPassword)
	if err != nil {
		return err
	}
	u.Hash = hash
	// Anytime a temporary password is created, we will force the user
	// to change their password
	u.PasswordChangeRequired = true
	err = db.Save(u).Error
	if err != nil {
		return err
	}
	log.Infof("Please login with the username admin and the password %s", temporaryPassword)
	return nil
}

// Setup initializes the database and runs any needed migrations.
//
// First, it establishes a connection to the database, then runs any migrations
// newer than the version the database is on.
//
// Once the database is up-to-date, we create an admin user (if needed) that
// has a randomly generated API key and password.
func Setup(c *config.Config) error {
	// Setup the package-scoped config
	conf = c
	// Setup the goose configuration
	migrateConf := &goose.DBConf{
		MigrationsDir: conf.MigrationsPath,
		Env:           "production",
		Driver:        chooseDBDriver(conf.DBName, conf.DBPath),
	}
	// Get the latest possible migration
	latest, err := goose.GetMostRecentDBVersion(migrateConf.MigrationsDir)
	if err != nil {
		log.Error(err)
		return err
	}

	// Register certificates for tls encrypted db connections
	if conf.DBSSLCaPath != "" {
		switch conf.DBName {
		case "mysql":
			rootCertPool := x509.NewCertPool()
			pem, err := ioutil.ReadFile(conf.DBSSLCaPath)
			if err != nil {
				log.Error(err)
				return err
			}
			if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
				log.Error("Failed to append PEM.")
				return err
			}
			mysql.RegisterTLSConfig("ssl_ca", &tls.Config{
				RootCAs: rootCertPool,
			})
			// Default database is sqlite3, which supports no tls, as connection
			// is file based
		default:
		}
	}

	// Open our database connection using GORM v2 drivers
	i := 0
	for {
		var gormConfig *gorm.Config
		if conf.DBName == "mysql" {
			gormConfig = &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			}
			db, err = gorm.Open(gormmysql.Open(getDBConnectionString(conf)), gormConfig)
		} else {
			gormConfig = &gorm.Config{
				Logger: logger.Default.LogMode(logger.Silent),
			}
			db, err = gorm.Open(sqlite.Open(getDBConnectionString(conf)), gormConfig)
		}
		if err == nil {
			break
		}
		if err != nil && i >= MaxDatabaseConnectionAttempts {
			log.Error(err)
			return err
		}
		i += 1
		log.Warn("waiting for database to be up...")
		time.Sleep(5 * time.Second)
	}

	// Get the underlying sql.DB for goose migrations and pool settings
	sqlDB, err := db.DB()
	if err != nil {
		log.Error(err)
		return err
	}
	sqlDB.SetMaxOpenConns(1)

	// For SQLite on-disk databases, checkpoint any leftover WAL from a previous
	// run and set up read-write splitting. If the process was killed with
	// SIGKILL, CloseDB() never ran and the WAL file may contain unflushed data.
	// The checkpoint runs before opening the reader connection so no other
	// readers exist that could block TRUNCATE.
	// In-memory databases (used in tests) must NOT create a reader because each
	// connection creates an independent :memory: database, causing goose
	// migration tables to be invisible to subsequent queries.
	if isOnDiskSQLite() {
		// Startup WAL checkpoint
		if _, err := sqlDB.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
			log.Warnf("Startup WAL checkpoint failed (non-fatal): %v", err)
		} else {
			log.Info("Startup WAL checkpoint completed")
		}

		// Open a separate reader connection for read-write splitting.
		// This avoids any hidden connection pool plugins — the reader is a
		// plain gorm.DB with its own *sql.DB pool.
		readDSN := getDBConnectionString(conf)
		readerDB, err = gorm.Open(sqlite.Open(readDSN), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			log.Errorf("failed to open reader database: %v", err)
			return err
		}
		rSqlDB, err := readerDB.DB()
		if err != nil {
			log.Errorf("failed to get reader sql.DB: %v", err)
			return err
		}
		maxConns := conf.DBReaderMaxConns
		if maxConns <= 0 {
			maxConns = defaultReaderMaxConns
		}
		rSqlDB.SetMaxOpenConns(maxConns)
		log.Infof("SQLite read-write splitting enabled (reader max_conns=%d)", maxConns)
	}

	// Migrate up to the latest version
	err = goose.RunMigrationsOnDb(migrateConf, migrateConf.MigrationsDir, latest, sqlDB)
	if err != nil {
		log.Error(err)
		return err
	}
	// Create the admin user if it doesn't exist
	var userCount int64
	var adminUser User
	db.Model(&User{}).Count(&userCount)
	adminRole, err := GetRoleBySlug(RoleAdmin)
	if err != nil {
		log.Error(err)
		return err
	}
	if userCount == 0 {
		adminUser := User{
			Username:               DefaultAdminUsername,
			Role:                   adminRole,
			RoleID:                 adminRole.ID,
			PasswordChangeRequired: true,
		}

		if envToken := os.Getenv(InitialAdminApiToken); envToken != "" {
			adminUser.ApiKey = envToken
		} else {
			adminUser.ApiKey = auth.GenerateSecureKey(auth.APIKeyLength)
		}

		err = db.Save(&adminUser).Error
		if err != nil {
			log.Error(err)
			return err
		}
	}
	// If this is the first time the user is installing Gophish, then we will
	// generate a temporary password for the admin user.
	//
	// We do this here instead of in the block above where the admin is created
	// since there's the chance the user executes Gophish and has some kind of
	// error, then tries restarting it. If they didn't grab the password out of
	// the logs, then they would have lost it.
	//
	// By doing the temporary password here, we will regenerate that temporary
	// password until the user is able to reset the admin password.
	if adminUser.Username == "" {
		adminUser, err = GetUserByUsername(DefaultAdminUsername)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	if adminUser.PasswordChangeRequired {
		err = createTemporaryPassword(&adminUser)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}
