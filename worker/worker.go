package worker

import (
	"context"
	"fmt"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/mailer"
	"github.com/gophish/gophish/models"
	"github.com/sirupsen/logrus"
)

// Worker is an interface that defines the operations needed for a background worker
type Worker interface {
	Start()
	LaunchCampaign(c models.Campaign)
	SendTestEmail(s *models.EmailRequest) error
}

// DefaultWorker is the background worker that handles watching for new campaigns and sending emails appropriately.
type DefaultWorker struct {
	mailer mailer.Mailer
}

// New creates a new worker object to handle the creation of campaigns
func New(options ...func(Worker) error) (Worker, error) {
	defaultMailer := mailer.NewMailWorker()
	w := &DefaultWorker{
		mailer: defaultMailer,
	}
	for _, opt := range options {
		if err := opt(w); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// WithMailer sets the mailer for a given worker.
// By default, workers use a standard, default mailworker.
func WithMailer(m mailer.Mailer) func(*DefaultWorker) error {
	return func(w *DefaultWorker) error {
		w.mailer = m
		return nil
	}
}

// processCampaigns loads maillogs scheduled to be sent before the provided
// time and sends them to the mailer.
func (w *DefaultWorker) processCampaigns(t time.Time) error {
	ms, err := models.GetQueuedMailLogs(t.UTC())
	if err != nil {
		log.Error(err)
		return err
	}
	// Lock the MailLogs (they will be unlocked after processing)
	err = models.LockMailLogs(ms, true)
	if err != nil {
		return err
	}
	campaignCache := make(map[int64]models.Campaign)
	// smtpIdCache maps campaignId → {rid → smtpId} for batch lookups.
	smtpIdCache := make(map[int64]map[string]int64)
	// We group the maillogs by (campaignId, smtpId) so that each batch
	// can be sent through a single SMTP connection.  The composite key
	// is "campaignId:smtpId".
	msg := make(map[string][]mailer.Mail)
	for _, m := range ms {
		// We cache the campaign here to greatly reduce the time it takes to
		// generate the message (ref #1726)
		c, ok := campaignCache[m.CampaignId]
		if !ok {
			c, err = models.GetCampaignMailContext(m.CampaignId, m.UserId)
			if err != nil {
				return err
			}
			campaignCache[c.Id] = c
		}
		m.CacheCampaign(&c)

		// Determine the SMTPId for this maillog's recipient.
		smtpIdMap, ok := smtpIdCache[m.CampaignId]
		if !ok {
			smtpIdMap, err = models.GetResultSMTPIdMap(m.CampaignId)
			if err != nil {
				log.Warn(err)
				smtpIdMap = map[string]int64{}
			}
			smtpIdCache[m.CampaignId] = smtpIdMap
		}
		smtpId := smtpIdMap[m.RId]
		if smtpId == 0 {
			smtpId = c.SMTPId // legacy fallback
		}
		key := fmt.Sprintf("%d:%d", m.CampaignId, smtpId)
		msg[key] = append(msg[key], m)
	}

	// Track which campaigns we've already marked as In-progress to
	// avoid redundant status updates when multiple SMTP groups belong
	// to the same campaign.
	campaignStarted := make(map[int64]bool)

	// Next, we process each group of maillogs in parallel
	for key, msc := range msg {
		// Extract campaignId from the first entry in the group.
		firstML := msc[0].(*models.MailLog)
		cid := firstML.CampaignId
		c := campaignCache[cid]
		if c.Status == models.CampaignQueued && !campaignStarted[cid] {
			campaignStarted[cid] = true
			err := c.UpdateStatus(models.CampaignInProgress)
			if err != nil {
				log.Error(err)
				return err
			}
		}
		go func(key string, msc []mailer.Mail) {
			log.WithFields(logrus.Fields{
				"num_emails": len(msc),
				"group_key":  key,
			}).Info("Sending emails to mailer for processing")
			w.mailer.Queue(msc)
		}(key, msc)
	}
	return nil
}

// Start launches the worker to poll the database every minute for any pending maillogs
// that need to be processed.
func (w *DefaultWorker) Start() {
	log.Info("Background Worker Started Successfully - Waiting for Campaigns")
	go w.mailer.Start(context.Background())
	for t := range time.Tick(1 * time.Minute) {
		err := w.processCampaigns(t)
		if err != nil {
			log.Error(err)
			continue
		}
	}
}

// LaunchCampaign starts a campaign
func (w *DefaultWorker) LaunchCampaign(c models.Campaign) {
	ms, err := models.GetMailLogsByCampaign(c.Id)
	if err != nil {
		log.Error(err)
		return
	}
	models.LockMailLogs(ms, true)
	currentTime := time.Now().UTC()
	campaignMailCtx, err := models.GetCampaignMailContext(c.Id, c.UserId)
	if err != nil {
		log.Error(err)
		return
	}
	// Batch-fetch SMTPId mapping for all results in this campaign.
	smtpIdMap, err := models.GetResultSMTPIdMap(c.Id)
	if err != nil {
		log.Warn(err)
		smtpIdMap = map[string]int64{}
	}
	// Group mail entries by (campaignId, smtpId).
	grouped := make(map[string][]mailer.Mail)
	for _, m := range ms {
		// Only send the emails scheduled to be sent for the past minute to
		// respect the campaign scheduling options
		if m.SendDate.After(currentTime) {
			m.Unlock()
			continue
		}
		err = m.CacheCampaign(&campaignMailCtx)
		if err != nil {
			log.Error(err)
			return
		}
		smtpId := smtpIdMap[m.RId]
		if smtpId == 0 {
			smtpId = campaignMailCtx.SMTPId
		}
		key := fmt.Sprintf("%d:%d", c.Id, smtpId)
		grouped[key] = append(grouped[key], m)
	}
	for _, mailEntries := range grouped {
		w.mailer.Queue(mailEntries)
	}
}

// SendTestEmail sends a test email
func (w *DefaultWorker) SendTestEmail(s *models.EmailRequest) error {
	go func() {
		ms := []mailer.Mail{s}
		w.mailer.Queue(ms)
	}()
	return <-s.ErrorChan
}
