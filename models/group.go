package models

import (
	"errors"
	"fmt"
	"net/mail"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
)

// Group contains the fields needed for a user -> group mapping
// Groups contain 1..* Targets
type Group struct {
	Id           int64     `json:"id"`
	UserId       int64     `json:"-"`
	Name         string    `json:"name"`
	ModifiedDate time.Time `json:"modified_date"`
	Targets      []Target  `json:"targets" sql:"-"`
}

// GroupSummaries is a struct representing the overview of Groups.
type GroupSummaries struct {
	Total  int64          `json:"total"`
	Groups []GroupSummary `json:"groups"`
}

// GroupSummary represents a summary of the Group model. The only
// difference is that, instead of listing the Targets (which could be expensive
// for large groups), it lists the target count.
type GroupSummary struct {
	Id           int64     `json:"id"`
	Name         string    `json:"name"`
	ModifiedDate time.Time `json:"modified_date"`
	NumTargets   int64     `json:"num_targets"`
}

// GroupTarget is used for a many-to-many relationship between 1..* Groups and 1..* Targets
type GroupTarget struct {
	GroupId  int64 `json:"-"`
	TargetId int64 `json:"-"`
}

// Target contains the fields needed for individual targets specified by the user
// Groups contain 1..* Targets, but 1 Target may belong to 1..* Groups
type Target struct {
	Id int64 `json:"-"`
	BaseRecipient
}

// BaseRecipient contains the fields for a single recipient. This is the base
// struct used in members of groups and campaign results.
type BaseRecipient struct {
	Email    string `json:"email"`
	FullName string `json:"full_name"`
	Position string `json:"position"`
}

// FormatAddress returns the email address to use in the "To" header of the email
func (r *BaseRecipient) FormatAddress() string {
	addr := r.Email
	if r.FullName != "" {
		a := &mail.Address{
			Name:    r.FullName,
			Address: r.Email,
		}
		addr = a.String()
	}
	return addr
}

// ErrEmailNotSpecified is thrown when no email is specified for the Target
var ErrEmailNotSpecified = errors.New("No email address specified")

// ErrGroupNameNotSpecified is thrown when a group name is not specified
var ErrGroupNameNotSpecified = errors.New("Group name not specified")

// ErrNoTargetsSpecified is thrown when no targets are specified by the user
var ErrNoTargetsSpecified = errors.New("No targets specified")

// Validate performs validation on a group given by the user
func (g *Group) Validate() error {
	switch {
	case g.Name == "":
		return ErrGroupNameNotSpecified
	case len(g.Targets) == 0:
		return ErrNoTargetsSpecified
	}
	return nil
}

// GetGroups returns the groups owned by the given user.
func GetGroups(uid int64, pp PageParams) ([]Group, int64, error) {
	gs := []Group{}
	var total int64
	if pp.Valid() {
		if err := db.Table("groups").Where("user_id=?", uid).Count(&total).Error; err != nil {
			log.Error(err)
			return gs, 0, err
		}
	}
	query := db.Where("user_id=?", uid).Order("modified_date DESC")
	if pp.Valid() {
		query = query.Limit(pp.PageSize).Offset(pp.Offset())
	}
	err := query.Find(&gs).Error
	if err != nil {
		log.Error(err)
		return gs, 0, err
	}
	for i := range gs {
		gs[i].Targets, err = GetTargets(gs[i].Id)
		if err != nil {
			log.Error(err)
		}
	}
	if !pp.Valid() {
		total = int64(len(gs))
	}
	return gs, total, nil
}

// GetGroupSummaries returns the summaries for the groups
// created by the given uid.
func GetGroupSummaries(uid int64) (GroupSummaries, error) {
	gs := GroupSummaries{}
	query := db.Table("groups").Where("user_id=?", uid)
	err := query.Select("id, name, modified_date").Scan(&gs.Groups).Error
	if err != nil {
		log.Error(err)
		return gs, err
	}
	for i := range gs.Groups {
		query = db.Table("group_targets").Where("group_id=?", gs.Groups[i].Id)
		err = query.Count(&gs.Groups[i].NumTargets).Error
		if err != nil {
			return gs, err
		}
	}
	gs.Total = int64(len(gs.Groups))
	return gs, nil
}

// GetGroup returns the group, if it exists, specified by the given id and user_id.
func GetGroup(id int64, uid int64) (Group, error) {
	g := Group{}
	err := db.Where("user_id=? and id=?", uid, id).Find(&g).Error
	if err != nil {
		log.Error(err)
		return g, err
	}
	g.Targets, err = GetTargets(g.Id)
	if err != nil {
		log.Error(err)
	}
	return g, nil
}

// GetGroupSummary returns the summary for the requested group
func GetGroupSummary(id int64, uid int64) (GroupSummary, error) {
	g := GroupSummary{}
	query := db.Table("groups").Where("user_id=? and id=?", uid, id)
	err := query.Select("id, name, modified_date").Scan(&g).Error
	if err != nil {
		log.Error(err)
		return g, err
	}
	query = db.Table("group_targets").Where("group_id=?", id)
	err = query.Count(&g.NumTargets).Error
	if err != nil {
		return g, err
	}
	return g, nil
}

// GetGroupByName returns the group, if it exists, specified by the given name and user_id.
func GetGroupByName(n string, uid int64) (Group, error) {
	g := Group{}
	err := db.Where("user_id=? and name=?", uid, n).Find(&g).Error
	if err != nil {
		log.Error(err)
		return g, err
	}
	g.Targets, err = GetTargets(g.Id)
	if err != nil {
		log.Error(err)
	}
	return g, err
}

// PostGroup creates a new group in the database.
func PostGroup(g *Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	// Insert the group into the DB
	tx := db.Begin()
	err := tx.Save(g).Error
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = insertTargetsIntoGroup(tx, g.Targets, g.Id)
	if err != nil {
		tx.Rollback()
		log.Error(err)
		return err
	}
	err = tx.Commit().Error
	if err != nil {
		log.Error(err)
		tx.Rollback()
		return err
	}
	return nil
}

// PutGroup updates the given group if found in the database.
func PutGroup(g *Group) error {
	if err := g.Validate(); err != nil {
		return err
	}
	// Fetch group's existing targets from database.
	ts, err := GetTargets(g.Id)
	if err != nil {
		log.WithFields(logrus.Fields{
			"group_id": g.Id,
		}).Error("Error getting targets from group")
		return err
	}
	// Preload the caches
	cacheNew := make(map[string]int64, len(g.Targets))
	for _, t := range g.Targets {
		cacheNew[t.Email] = t.Id
	}

	cacheExisting := make(map[string]int64, len(ts))
	for _, t := range ts {
		cacheExisting[t.Email] = t.Id
	}

	tx := db.Begin()
	// Check existing targets, removing any that are no longer in the group.
	for _, t := range ts {
		if _, ok := cacheNew[t.Email]; ok {
			continue
		}

		// If the target does not exist in the group any longer, we delete it
		err := tx.Where("group_id=? and target_id=?", g.Id, t.Id).Delete(&GroupTarget{}).Error
		if err != nil {
			tx.Rollback()
			log.WithFields(logrus.Fields{
				"email": t.Email,
			}).Error("Error deleting email")
		}
	}
	// Add any targets that are not in the database yet.
	var newTargets []Target
	for _, nt := range g.Targets {
		// If the target already exists in the database, we should just update
		// the record with the latest information.
		if id, ok := cacheExisting[nt.Email]; ok {
			nt.Id = id
			err = UpdateTarget(tx, nt)
			if err != nil {
				log.Error(err)
				tx.Rollback()
				return err
			}
			continue
		}
		// Otherwise, add target if not in database
		newTargets = append(newTargets, nt)
	}
	if len(newTargets) > 0 {
		err = insertTargetsIntoGroup(tx, newTargets, g.Id)
		if err != nil {
			log.Error(err)
			tx.Rollback()
			return err
		}
	}
	err = tx.Save(g).Error
	if err != nil {
		log.Error(err)
		return err
	}
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

// DeleteGroup deletes a given group by group ID and user ID
func DeleteGroup(g *Group) error {
	// Delete all the group_targets entries for this group
	err := db.Where("group_id=?", g.Id).Delete(&GroupTarget{}).Error
	if err != nil {
		log.Error(err)
		return err
	}
	// Delete the group itself
	err = db.Delete(g).Error
	if err != nil {
		log.Error(err)
		return err
	}
	return err
}

// insertTargetsIntoGroup imports the given targets into the group, preserving
// the previous FirstOrCreate semantics (a target is only reused if the email,
// full name and position all match). It replaces one round-trip per target with
// a few batched queries: a single lookup of existing targets, one (chunked)
// INSERT for new targets, and one (chunked) INSERT for the group mappings.
func insertTargetsIntoGroup(tx *gorm.DB, targets []Target, gid int64) error {
	if len(targets) == 0 {
		return nil
	}
	type rec struct {
		Id       int64
		Email    string
		FullName string
		Position string
	}
	keyOf := func(email, fullName, position string) string {
		return email + "\x00" + fullName + "\x00" + position
	}
	// Validate every email up front, matching the previous behavior of stopping
	// on the first invalid address (which aborts the whole transaction).
	emails := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		if _, err := mail.ParseAddress(t.Email); err != nil {
			log.WithFields(logrus.Fields{"email": t.Email}).Error("Invalid email")
			return err
		}
		emails[t.Email] = struct{}{}
	}
	emailList := make([]string, 0, len(emails))
	for e := range emails {
		emailList = append(emailList, e)
	}
	// Existing targets keyed by (email, full name, position). If multiple rows
	// match, keep the lowest id to stay consistent with gorm's First() (lowest
	// primary key).
	existing := make(map[string]int64)
	existingRows := []rec{}
	for _, chunk := range chunkStrings(emailList) {
		rows := []rec{}
		if err := tx.Table("targets").
			Select("id, email, full_name, position").
			Where("email IN (?)", chunk).Scan(&rows).Error; err != nil {
			log.Error(err)
			return err
		}
		existingRows = append(existingRows, rows...)
	}
	for _, r := range existingRows {
		k := keyOf(r.Email, r.FullName, r.Position)
		if id, ok := existing[k]; !ok || r.Id < id {
			existing[k] = r.Id
		}
	}
	// Determine which targets need to be newly created.
	newByKey := map[string]Target{}
	for _, t := range targets {
		k := keyOf(t.Email, t.FullName, t.Position)
		if _, ok := existing[k]; ok {
			continue
		}
		newByKey[k] = t
	}
	// Insert the new targets, then read back their assigned ids.
	if len(newByKey) > 0 {
		if err := insertTargetsBulk(tx, newByKey); err != nil {
			return err
		}
		newEmails := make([]string, 0, len(newByKey))
		for _, t := range newByKey {
			newEmails = append(newEmails, t.Email)
		}
		for _, chunk := range chunkStrings(newEmails) {
			resolved := []rec{}
			if err := tx.Table("targets").
				Select("id, email, full_name, position").
				Where("email IN (?)", chunk).Scan(&resolved).Error; err != nil {
				log.Error(err)
				return err
			}
			for _, r := range resolved {
				k := keyOf(r.Email, r.FullName, r.Position)
				if _, ok := newByKey[k]; ok {
					existing[k] = r.Id
				}
			}
		}
	}
	// Build the group_targets mappings, one per input target (duplicates produce
	// duplicate mappings, matching the previous per-target behavior).
	mapping := make([]int64, 0, len(targets))
	for _, t := range targets {
		k := keyOf(t.Email, t.FullName, t.Position)
		id, ok := existing[k]
		if !ok {
			return fmt.Errorf("target %q missing after import", t.Email)
		}
		mapping = append(mapping, id)
	}
	return insertGroupTargetsBulk(tx, gid, mapping)
}

// chunkStrings splits a slice of values into chunks that stay well below
// SQLite's per-statement bind variable limit (999 for SQLite 3.31).
func chunkStrings(vals []string) [][]string {
	const maxVars = 800
	chunks := [][]string{}
	for start := 0; start < len(vals); start += maxVars {
		end := start + maxVars
		if end > len(vals) {
			end = len(vals)
		}
		chunks = append(chunks, vals[start:end])
	}
	return chunks
}

// insertTargetsBulk inserts the newly created targets in chunks, keeping the
// number of bind variables within SQLite's per-statement limit.
func insertTargetsBulk(tx *gorm.DB, byKey map[string]Target) error {
	keys := make([]string, 0, len(byKey))
	for k := range byKey {
		keys = append(keys, k)
	}
	const batchSize = 300
	for start := 0; start < len(keys); start += batchSize {
		end := start + batchSize
		if end > len(keys) {
			end = len(keys)
		}
		sql := "INSERT INTO targets (email, full_name, position) VALUES"
		var args []interface{}
		for i := start; i < end; i++ {
			if i > start {
				sql += ","
			}
			sql += " (?,?,?)"
			t := byKey[keys[i]]
			args = append(args, t.Email, t.FullName, t.Position)
		}
		if err := tx.Exec(sql, args...).Error; err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// insertGroupTargetsBulk inserts the many-to-many group mappings in chunks.
func insertGroupTargetsBulk(tx *gorm.DB, gid int64, targetIds []int64) error {
	const batchSize = 300
	for start := 0; start < len(targetIds); start += batchSize {
		end := start + batchSize
		if end > len(targetIds) {
			end = len(targetIds)
		}
		sql := "INSERT INTO group_targets (group_id, target_id) VALUES"
		var args []interface{}
		for i := start; i < end; i++ {
			if i > start {
				sql += ","
			}
			sql += " (?,?)"
			args = append(args, gid, targetIds[i])
		}
		if err := tx.Exec(sql, args...).Error; err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

// UpdateTarget updates the given target information in the database.
func UpdateTarget(tx *gorm.DB, target Target) error {
	targetInfo := map[string]interface{}{
		"full_name": target.FullName,
		"position":  target.Position,
	}
	err := tx.Model(&target).Where("id = ?", target.Id).Updates(targetInfo).Error
	if err != nil {
		log.WithFields(logrus.Fields{
			"email": target.Email,
		}).Error("Error updating target information")
	}
	return err
}

// GetTargets performs a many-to-many select to get all the Targets for a Group
func GetTargets(gid int64) ([]Target, error) {
	ts := []Target{}
	err := db.Table("targets").Select("targets.id, targets.email, targets.full_name, targets.position").Joins("left join group_targets gt ON targets.id = gt.target_id").Where("gt.group_id=?", gid).Scan(&ts).Error
	return ts, err
}
