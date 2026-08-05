package models

import (
	check "gopkg.in/check.v1"
)

func (s *ModelsSuite) TestCampaignSMTPTableName(c *check.C) {
	cs := CampaignSMTP{}
	c.Assert(cs.TableName(), check.Equals, "campaign_smtps")
}

func (s *ModelsSuite) TestGetCampaignSMTPsEmpty(c *check.C) {
	csmtps, err := GetCampaignSMTPs(999)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(csmtps), check.Equals, 0)
}

func (s *ModelsSuite) TestPostAndDeleteCampaignSMTPs(c *check.C) {
	// Create two SMTP profiles
	smtp1 := SMTP{Name: "SMTP1", Host: "1.1.1.1:25", FromAddress: "a@test.com", UserId: 1}
	c.Assert(PostSMTP(&smtp1), check.Equals, nil)
	smtp2 := SMTP{Name: "SMTP2", Host: "2.2.2.2:25", FromAddress: "b@test.com", UserId: 1}
	c.Assert(PostSMTP(&smtp2), check.Equals, nil)

	// Create a campaign with dependencies
	cd := s.createCampaignDependencies(c)
	c.Assert(PostCampaign(&cd, cd.UserId), check.Equals, nil)

	// Post CampaignSMTP records
	err := PostCampaignSMTPs(db, cd.Id, []int64{smtp1.Id, smtp2.Id})
	c.Assert(err, check.Equals, nil)

	// Verify
	csmtps, err := GetCampaignSMTPs(cd.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(csmtps), check.Equals, 2)
	c.Assert(csmtps[0].SMTPId, check.Equals, smtp1.Id)
	c.Assert(csmtps[0].Position, check.Equals, 0)
	c.Assert(csmtps[1].SMTPId, check.Equals, smtp2.Id)
	c.Assert(csmtps[1].Position, check.Equals, 1)

	// Delete
	err = DeleteCampaignSMTPsByCampaign(cd.Id)
	c.Assert(err, check.Equals, nil)

	csmtps, err = GetCampaignSMTPs(cd.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(csmtps), check.Equals, 0)
}

func (s *ModelsSuite) TestPostCampaignSMTPsIdempotent(c *check.C) {
	smtp1 := SMTP{Name: "SMTP1", Host: "1.1.1.1:25", FromAddress: "a@test.com", UserId: 1}
	c.Assert(PostSMTP(&smtp1), check.Equals, nil)
	smtp2 := SMTP{Name: "SMTP2", Host: "2.2.2.2:25", FromAddress: "b@test.com", UserId: 1}
	c.Assert(PostSMTP(&smtp2), check.Equals, nil)

	cd := s.createCampaignDependencies(c)
	c.Assert(PostCampaign(&cd, cd.UserId), check.Equals, nil)

	// First post
	err := PostCampaignSMTPs(db, cd.Id, []int64{smtp1.Id})
	c.Assert(err, check.Equals, nil)

	// Second post with different SMTPs — should replace, not duplicate
	err = PostCampaignSMTPs(db, cd.Id, []int64{smtp1.Id, smtp2.Id})
	c.Assert(err, check.Equals, nil)

	csmtps, err := GetCampaignSMTPs(cd.Id)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(csmtps), check.Equals, 2)
}

func (s *ModelsSuite) TestGetCampaignSMTPRecords(c *check.C) {
	smtp1 := SMTP{Name: "SMTP1", Host: "1.1.1.1:25", FromAddress: "a@test.com", UserId: 1}
	c.Assert(PostSMTP(&smtp1), check.Equals, nil)

	cd := s.createCampaignDependencies(c)
	c.Assert(PostCampaign(&cd, cd.UserId), check.Equals, nil)

	err := PostCampaignSMTPs(db, cd.Id, []int64{smtp1.Id})
	c.Assert(err, check.Equals, nil)

	records, err := GetCampaignSMTPRecords(cd.Id, 1)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(records), check.Equals, 1)
	c.Assert(records[0].Name, check.Equals, "SMTP1")
}

func (s *ModelsSuite) TestGetCampaignSMTPRecordsWrongUser(c *check.C) {
	smtp1 := SMTP{Name: "SMTP1", Host: "1.1.1.1:25", FromAddress: "a@test.com", UserId: 1}
	c.Assert(PostSMTP(&smtp1), check.Equals, nil)

	cd := s.createCampaignDependencies(c)
	c.Assert(PostCampaign(&cd, cd.UserId), check.Equals, nil)

	err := PostCampaignSMTPs(db, cd.Id, []int64{smtp1.Id})
	c.Assert(err, check.Equals, nil)

	// Query with wrong user — should return empty since SMTP belongs to user 1
	records, err := GetCampaignSMTPRecords(cd.Id, 999)
	c.Assert(err, check.Equals, nil)
	c.Assert(len(records), check.Equals, 0)
}

func (s *ModelsSuite) TestDeleteCampaignSMTPsByCampaignEmpty(c *check.C) {
	// Deleting non-existent records should not error
	err := DeleteCampaignSMTPsByCampaign(999)
	c.Assert(err, check.Equals, nil)
}
