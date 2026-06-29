package models

import (
	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// CampaignSMTP is a join table that maps a campaign to multiple sending
// profiles (SMTP). The Position field records the order in which profiles
// were selected by the user and is used for interval-based even distribution:
//
//	base = total / numProfiles, remainder = total % numProfiles
//	first `remainder` profiles each get (base+1), the rest get `base`.
//
// This table is intentionally kept separate from the campaigns table so
// that the original single-SMTPId column can remain for backward
// compatibility with older API clients and data.
type CampaignSMTP struct {
	Id         int64 `json:"id" gorm:"column:id; primary_key:yes"`
	CampaignId int64 `json:"campaign_id" gorm:"column:campaign_id"`
	SMTPId     int64 `json:"smtp_id" gorm:"column:smtp_id"`
	Position   int   `json:"position" gorm:"column:position"`
}

// TableName specifies the database table name for Gorm.
func (CampaignSMTP) TableName() string {
	return "campaign_smtps"
}

// GetCampaignSMTPs returns all CampaignSMTP records for the given campaign,
// ordered by Position.
func GetCampaignSMTPs(campaignId int64) ([]CampaignSMTP, error) {
	csmtps := []CampaignSMTP{}
	err := db.Where("campaign_id = ?", campaignId).Order("position ASC").Find(&csmtps).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Error(err)
		return csmtps, err
	}
	return csmtps, nil
}

// PostCampaignSMTPs inserts a batch of CampaignSMTP records inside a
// transaction. It first deletes any existing records for the campaign to
// allow idempotent updates.
func PostCampaignSMTPs(tx *gorm.DB, campaignId int64, smtpIds []int64) error {
	// Remove old mappings
	err := tx.Where("campaign_id = ?", campaignId).Delete(&CampaignSMTP{}).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	for i, smtpId := range smtpIds {
		cs := &CampaignSMTP{
			CampaignId: campaignId,
			SMTPId:     smtpId,
			Position:   i,
		}
		if err := tx.Save(cs).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetCampaignSMTPRecords returns the full SMTP records (with Headers loaded)
// associated with the given campaign, ordered by position.
func GetCampaignSMTPRecords(campaignId int64) ([]SMTP, error) {
	csmtps, err := GetCampaignSMTPs(campaignId)
	if err != nil {
		return nil, err
	}
	smtps := make([]SMTP, 0, len(csmtps))
	for _, cs := range csmtps {
		s := SMTP{}
		err := db.Where("id=?", cs.SMTPId).Find(&s).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return smtps, err
		}
		err = db.Where("smtp_id=?", s.Id).Find(&s.Headers).Error
		if err != nil && err != gorm.ErrRecordNotFound {
			return smtps, err
		}
		smtps = append(smtps, s)
	}
	return smtps, nil
}

// DeleteCampaignSMTPsByCampaign removes all CampaignSMTP records for a
// given campaign.
func DeleteCampaignSMTPsByCampaign(campaignId int64) error {
	return db.Where("campaign_id = ?", campaignId).Delete(&CampaignSMTP{}).Error
}
