package app

import (
	"errors"
	"github.com/jinzhu/gorm"
	"strings"
	"time"
)

const revisionsTable = "revisions"

type revisionAPI struct {
	sqlDB *gorm.DB
}

var revisionManager *revisionAPI

// StartRevisionManager initializes revision manager
func StartRevisionManager(sqlDB *gorm.DB) {
	var err error
	switch {
	case sqlDB == nil:
		err = errors.New("sqlDB should not be nil")
	}
	handleError(err)

	// Auto migration
	revisionManager = &revisionAPI{sqlDB: sqlDB}
	handleError(sqlDB.AutoMigrate(&revision{}).Error)
}

type revision struct {
	Revision      int    `gorm:"type:tinyint(3);not null"`
	ResourceGroup string `gorm:"type:varchar(30);not null"`
	Data          []byte `gorm:"type:json;not null"`
	gorm.Model
}

func (revision *revision) validate() error {
	var err error
	switch {
	case revision.Revision == 0:
		err = errors.New("revision not specified")
	case strings.TrimSpace(revision.ResourceGroup) == "":
		err = errors.New("revision resource missing")
	case revision.Data == nil:
		err = errors.New("revision data missing")
	}
	return err
}

func (revisionAPI *revisionAPI) Add(rev *revision) error {
	err := rev.validate()
	if err != nil {
		return err
	}

	_, err = revisionAPI.Get(rev.ResourceGroup, rev.Revision)
	if !gorm.IsRecordNotFoundError(err) {
		// Update existing revision
		err = revisionAPI.sqlDB.Table(revisionsTable).Where("revision=? AND resource_group=?", rev.Revision, rev.ResourceGroup).
			Updates(rev).Error
		if err != nil {
			return err
		}
	} else {
		// Add new revision
		err = revisionAPI.sqlDB.Create(rev).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (revisionAPI *revisionAPI) Get(resource string, rev int) (*revision, error) {
	revision := &revision{}
	err := revisionAPI.sqlDB.First(revision, "revision=? AND resource_group=?", rev, resource).Error
	if err != nil {
		return nil, err
	}

	return revision, nil
}

func (revisionAPI *revisionAPI) List(resource string) ([]*revision, error) {
	revisions := make([]*revision, 0)
	err := revisionAPI.sqlDB.Table(revisionsTable).Order("revision, created_at").
		Find(&revisions, "resource_group = ?", resource).Error
	switch {
	case err == nil:
	default:
		return nil, err
	}
	return revisions, nil
}

func updateRevisionWorker(dur time.Duration, f func()) {
	f()
	for {
		select {
		case <-time.After(dur):
			f()
		}
	}
}
