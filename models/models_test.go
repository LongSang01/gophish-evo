package models

import (
	"testing"

	"github.com/gophish/gophish/config"
	"gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { check.TestingT(t) }

type ModelsSuite struct {
	config *config.Config
}

var _ = check.Suite(&ModelsSuite{})

func (s *ModelsSuite) SetUpSuite(c *check.C) {
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	s.config = conf
	err := Setup(conf)
	if err != nil {
		c.Fatalf("Failed creating database: %v", err)
	}
}

func (s *ModelsSuite) TearDownTest(c *check.C) {
	// Clear database tables between each test. If new tables are
	// used in this test suite they will need to be cleaned up here.
	// GORM v2 requires explicit WHERE clause for Delete to prevent
	// accidental full-table deletes.
	db.Where("1=1").Delete(Group{})
	db.Where("1=1").Delete(Target{})
	db.Where("1=1").Delete(GroupTarget{})
	db.Where("1=1").Delete(SMTP{})
	db.Where("1=1").Delete(Page{})
	db.Where("1=1").Delete(Template{})
	db.Where("1=1").Delete(Attachment{})
	db.Where("1=1").Delete(Result{})
	db.Where("1=1").Delete(MailLog{})
	db.Where("1=1").Delete(Campaign{})
	db.Where("1=1").Delete(CampaignSMTP{})
	db.Where("1=1").Delete(IMAP{})
	db.Where("1=1").Delete(Webhook{})
	db.Where("1=1").Delete(Event{})
	db.Where("1=1").Delete(EmailRequest{})
	db.Where("1=1").Delete(ReportExt{})
	db.Where("1=1").Delete(PageClickStats{})

	// Reset users table to default state.
	db.Where("id != 1").Delete(User{})
	db.Model(User{}).Where("1=1").Update("username", "admin")
	// NOTE: Permission and Role tables are NOT cleared — they contain
	// static seed data inserted by migrations and must persist.
}

func (s *ModelsSuite) createCampaignDependencies(ch *check.C, optional ...string) Campaign {
	// we use the optional parameter to pass an alternative subject
	group := Group{Name: "Test Group"}
	group.Targets = []Target{
		Target{BaseRecipient: BaseRecipient{Email: "test1@example.com", FullName: "First Example"}},
		Target{BaseRecipient: BaseRecipient{Email: "test2@example.com", FullName: "Second Example"}},
		Target{BaseRecipient: BaseRecipient{Email: "test3@example.com", FullName: "Second Example"}},
		Target{BaseRecipient: BaseRecipient{Email: "test4@example.com", FullName: "Second Example"}},
	}
	group.UserId = 1
	ch.Assert(PostGroup(&group), check.Equals, nil)

	// Add a template
	t := Template{Name: "Test Template"}
	if len(optional) > 0 {
		t.Subject = optional[0]
	} else {
		t.Subject = "{{.RId}} - Subject"
	}
	t.Text = "{{.RId}} - Text"
	t.HTML = "{{.RId}} - HTML"
	t.UserId = 1
	ch.Assert(PostTemplate(&t), check.Equals, nil)

	// Add a landing page
	p := Page{Name: "Test Page"}
	p.HTML = "<html>Test</html>"
	p.UserId = 1
	ch.Assert(PostPage(&p), check.Equals, nil)

	// Add a sending profile
	smtp := SMTP{Name: "Test Page"}
	smtp.UserId = 1
	smtp.Host = "example.com"
	smtp.FromAddress = "test@test.com"
	ch.Assert(PostSMTP(&smtp), check.Equals, nil)

	c := Campaign{Name: "Test campaign"}
	c.UserId = 1
	c.Template = t
	c.Page = p
	c.SMTP = smtp
	c.Groups = []Group{group}
	return c
}

func (s *ModelsSuite) createCampaign(ch *check.C) Campaign {
	c := s.createCampaignDependencies(ch)
	// Setup and "launch" our campaign
	ch.Assert(PostCampaign(&c, c.UserId), check.Equals, nil)

	// For comparing the dates, we need to fetch the campaign again. This is
	// to solve an issue where the campaign object right now has time down to
	// the microsecond, while in MySQL it's rounded down to the second.
	c, _ = GetCampaign(c.Id, c.UserId)
	return c
}

func setupBenchmark(b *testing.B) {
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	err := Setup(conf)
	if err != nil {
		b.Fatalf("Failed creating database: %v", err)
	}
}

func tearDownBenchmark(b *testing.B) {
	sqlDB, err := db.DB()
	if err != nil {
		b.Fatalf("error getting underlying sql.DB: %v", err)
	}
	err = sqlDB.Close()
	if err != nil {
		b.Fatalf("error closing database: %v", err)
	}
}

func resetBenchmark(b *testing.B) {
	db.Where("1=1").Delete(Group{})
	db.Where("1=1").Delete(Target{})
	db.Where("1=1").Delete(GroupTarget{})
	db.Where("1=1").Delete(SMTP{})
	db.Where("1=1").Delete(Page{})
	db.Where("1=1").Delete(Result{})
	db.Where("1=1").Delete(MailLog{})
	db.Where("1=1").Delete(Campaign{})

	// Reset users table to default state.
	db.Where("id != 1").Delete(User{})
	db.Model(User{}).Where("1=1").Update("username", "admin")
}
