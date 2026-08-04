package models

import (
	"errors"

	log "github.com/gophish/gophish/logger"
)

// Webhook represents the webhook model
type Webhook struct {
	Id       int64  `json:"id" gorm:"column:id; primary_key:yes"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Secret   string `json:"secret"`
	IsActive bool   `json:"is_active"`
}

// ErrURLNotSpecified indicates there was no URL specified
var ErrURLNotSpecified = errors.New("URL can't be empty")

// ErrNameNotSpecified indicates there was no name specified
var ErrNameNotSpecified = errors.New("Name can't be empty")

// GetWebhooks returns the webhooks
func GetWebhooks(pp PageParams) ([]Webhook, int64, error) {
	whs := []Webhook{}
	var total int64
	if pp.Valid() {
		if err := db.Table("webhooks").Count(&total).Error; err != nil {
			log.Error(err)
			return whs, 0, err
		}
	}
	query := db.Order("id DESC")
	if pp.Valid() {
		query = query.Limit(pp.PageSize).Offset(pp.Offset())
	}
	err := query.Find(&whs).Error
	if err != nil {
		log.Error(err)
		return whs, 0, err
	}
	if !pp.Valid() {
		total = int64(len(whs))
	}
	return whs, total, nil
}

// GetActiveWebhooks returns the active webhooks
func GetActiveWebhooks() ([]Webhook, error) {
	whs := []Webhook{}
	err := db.Where("is_active=?", true).Find(&whs).Error
	return whs, err
}

// GetWebhook returns the webhook that the given id corresponds to.
// If no webhook is found, an error is returned.
func GetWebhook(id int64) (Webhook, error) {
	wh := Webhook{}
	err := db.Where("id=?", id).First(&wh).Error
	return wh, err
}

// PostWebhook creates a new webhook in the database.
func PostWebhook(wh *Webhook) error {
	err := wh.Validate()
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Save(wh).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// PutWebhook edits an existing webhook in the database.
func PutWebhook(wh *Webhook) error {
	err := wh.Validate()
	if err != nil {
		log.Error(err)
		return err
	}
	err = db.Save(wh).Error
	return err
}

// DeleteWebhook deletes an existing webhook in the database.
// An error is returned if a webhook with the given id isn't found.
func DeleteWebhook(id int64) error {
	err := db.Where("id=?", id).Delete(&Webhook{}).Error
	return err
}

func (wh *Webhook) Validate() error {
	if wh.URL == "" {
		return ErrURLNotSpecified
	}
	if wh.Name == "" {
		return ErrNameNotSpecified
	}
	return nil
}
