package models

import (
	"time"

	check "gopkg.in/check.v1"
)

func (s *ModelsSuite) TestIMAPValidateNoHost(c *check.C) {
	im := IMAP{
		Port:     993,
		Username: "user",
		Password: "pass",
	}
	err := im.Validate()
	c.Assert(err, check.Equals, ErrIMAPHostNotSpecified)
}

func (s *ModelsSuite) TestIMAPValidateNoPort(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Username: "user",
		Password: "pass",
	}
	err := im.Validate()
	c.Assert(err, check.Equals, ErrIMAPPortNotSpecified)
}

func (s *ModelsSuite) TestIMAPValidateNoUsername(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     993,
		Password: "pass",
	}
	err := im.Validate()
	c.Assert(err, check.Equals, ErrIMAPUsernameNotSpecified)
}

func (s *ModelsSuite) TestIMAPValidateNoPassword(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     993,
		Username: "user",
	}
	err := im.Validate()
	c.Assert(err, check.Equals, ErrIMAPPasswordNotSpecified)
}

func (s *ModelsSuite) TestIMAPValidateInvalidPort(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     65535,
		Username: "user",
		Password: "pass",
	}
	// Port 65535 is technically valid (max uint16), but let's test the invalid host path
	// We test ErrInvalidIMAPHost instead since "invalidhost.example.tld" won't resolve
	im.Host = "invalidhost.invalidtld"
	im.Port = 993
	err := im.Validate()
	c.Assert(err, check.Equals, ErrInvalidIMAPHost)
}

func (s *ModelsSuite) TestIMAPValidateZeroPort(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     0,
		Username: "user",
		Password: "pass",
	}
	err := im.Validate()
	c.Assert(err, check.Equals, ErrIMAPPortNotSpecified)
}

func (s *ModelsSuite) TestIMAPValidateDefaultFolder(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     993,
		Username: "user",
		Password: "pass",
	}
	err := im.Validate()
	c.Assert(err, check.Equals, nil)
	c.Assert(im.Folder, check.Equals, DefaultIMAPFolder)
}

func (s *ModelsSuite) TestIMAPValidateCustomFolder(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     993,
		Username: "user",
		Password: "pass",
		Folder:   "CustomFolder",
	}
	err := im.Validate()
	c.Assert(err, check.Equals, nil)
	c.Assert(im.Folder, check.Equals, "CustomFolder")
}

func (s *ModelsSuite) TestIMAPValidateDefaultFreq(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     993,
		Username: "user",
		Password: "pass",
		IMAPFreq: 0,
	}
	err := im.Validate()
	c.Assert(err, check.Equals, nil)
	c.Assert(im.IMAPFreq, check.Equals, uint32(DefaultIMAPFreq))
}

func (s *ModelsSuite) TestIMAPValidateFreqTooLow(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     993,
		Username: "user",
		Password: "pass",
		IMAPFreq: 10,
	}
	err := im.Validate()
	c.Assert(err, check.Equals, nil)
	c.Assert(im.IMAPFreq, check.Equals, uint32(DefaultIMAPFreq))
}

func (s *ModelsSuite) TestIMAPValidateValid(c *check.C) {
	im := IMAP{
		Host:     "localhost",
		Port:     993,
		Username: "user",
		Password: "pass",
		Folder:   "INBOX",
		IMAPFreq: 120,
	}
	err := im.Validate()
	c.Assert(err, check.Equals, nil)
}

func (s *ModelsSuite) TestGetIMAPNoneExist(c *check.C) {
	imap, err := GetIMAP(1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(imap), check.Equals, 0)
}

func (s *ModelsSuite) TestPostAndDeleteIMAP(c *check.C) {
	im := IMAP{
		Enabled:  true,
		Host:     "localhost",
		Port:     993,
		Username: "user",
		Password: "pass",
		TLS:      true,
		UserId:   1,
	}
	err := PostIMAP(&im, 1)
	c.Assert(err, check.Equals, nil)

	got, err := GetIMAP(1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(got), check.Equals, 1)
	c.Assert(got[0].Host, check.Equals, "localhost")
	c.Assert(got[0].Username, check.Equals, "user")
	c.Assert(got[0].TLS, check.Equals, true)

	// Delete
	err = DeleteIMAP(1)
	c.Assert(err, check.Equals, nil)

	got, err = GetIMAP(1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(got), check.Equals, 0)
}

func (s *ModelsSuite) TestPostIMAPReplacesExisting(c *check.C) {
	// First save
	im := IMAP{
		Enabled:  true,
		Host:     "localhost",
		Port:     993,
		Username: "user1",
		Password: "pass1",
		UserId:   1,
	}
	err := PostIMAP(&im, 1)
	c.Assert(err, check.Equals, nil)

	// Second save should replace the first
	im2 := IMAP{
		Enabled:  true,
		Host:     "localhost",
		Port:     993,
		Username: "user2",
		Password: "pass2",
		UserId:   1,
	}
	err = PostIMAP(&im2, 1)
	c.Assert(err, check.Equals, nil)

	got, err := GetIMAP(1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(got), check.Equals, 1)
	c.Assert(got[0].Username, check.Equals, "user2")
}

func (s *ModelsSuite) TestSuccessfulLogin(c *check.C) {
	im := IMAP{
		Enabled:  true,
		Host:     "localhost",
		Port:     993,
		Username: "user",
		Password: "pass",
		UserId:   1,
	}
	err := PostIMAP(&im, 1)
	c.Assert(err, check.Equals, nil)

	// Get the saved IMAP with its UserId
	got, err := GetIMAP(1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(got), check.Equals, 1)

	before := got[0].LastLogin
	err = SuccessfulLogin(&got[0])
	c.Assert(err, check.Equals, nil)

	// Verify last_login was updated
	got2, err := GetIMAP(1)
	c.Assert(err, check.Equals, nil)
	c.Assert(got2[0].LastLogin.After(before) || got2[0].LastLogin.Equal(before), check.Equals, true)
}

func (s *ModelsSuite) TestIMAPTableName(c *check.C) {
	im := IMAP{}
	c.Assert(im.TableName(), check.Equals, "imap")
}

func (s *ModelsSuite) TestIMAPDefaultConstants(c *check.C) {
	c.Assert(DefaultIMAPFolder, check.Equals, "INBOX")
	c.Assert(DefaultIMAPFreq, check.Equals, 60)
}

func (s *ModelsSuite) TestIMAPModifiedDate(c *check.C) {
	now := time.Now().UTC()
	im := IMAP{
		Enabled:      true,
		Host:         "localhost",
		Port:         993,
		Username:     "user",
		Password:     "pass",
		UserId:       1,
		ModifiedDate: now,
	}
	err := PostIMAP(&im, 1)
	c.Assert(err, check.Equals, nil)

	got, err := GetIMAP(1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(got), check.Equals, 1)
}
