// Package backlab makes backing up your GitLab instance easy.
package backlab

import (
	"os"
	"path/filepath"

	"github.com/dchest/uniuri"
	"gopkg.in/kothar/go-backblaze.v0"
	"os/exec"
	"io/ioutil"
	"strconv"
	"time"
)

// Credentials is a type alias for backblaze.Credentials.
type Credentials backblaze.Credentials

// Config is a parameter used in Init for configuring backlab.
type Config struct {
	Credentials
	BucketName string
	// PreserveFor defines how long should the backups be kept in a bucket and locally. Doesn't delete any by default.
	PreserveFor int64
	BackupPath *string
}

// Backlab is a main struct.
type Backlab struct {
	Config

	b2 *backblaze.B2
}

// New configures connection with Backblaze and saves configuration to Backlab instance.
func New(config Config) (*Backlab, error) {
	b := &Backlab{
		Config: config,
	}

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      b.AccountID,
		ApplicationKey: b.ApplicationKey,
	})
	if err != nil {
		panic(err)
	}

	b.b2 = b2

	return b, nil
}

// Backup creates a new backup, removes old backups, and uploads the new backup to Backblaze
func (b *Backlab) Backup() {
	b.CreateBackup()
	b.RemoveOldLocalBackups()
}

// BackupArchive backups a file to Backblaze.
func (b *Backlab) BackupArchive(archivePath string) error {
	var (
		bucket *backblaze.Bucket
		err    error
	)
	if &b.BucketName == nil {
		randomString := uniuri.New()
		bucket, err = b.b2.CreateBucket("backlab-gitlab-backups-"+randomString, backblaze.AllPrivate)
	} else {
		bucket, err = b.b2.Bucket(b.BucketName)
	}
	if err != nil {
		return err
	}

	reader, _ := os.Open(archivePath)
	name := filepath.Base(archivePath)
	metadata := make(map[string]string)

	_, err = bucket.UploadFile(name, metadata, reader)
	if err != nil {
		return err
	}

	// scan bucket for backups older than specified and delete them

	return nil
}

// CreateBackup creates a local GitLab backup.
func (b *Backlab) CreateBackup() error {
	cmd := exec.Command("gitlab-rake", "gitlab:backup:create")
	err := cmd.Run()
	return err
}

// RemoveOldLocalBackups removes old local GitLab backups from BackupPath directory.
func (b *Backlab) RemoveOldLocalBackups() error {
	err := b.loopOverBackupFiles(func (f os.FileInfo, fp string, backupTimestamp int64) error {
		backupExpiryTimestamp := time.Now().Unix() - b.PreserveFor
		if backupTimestamp > backupExpiryTimestamp {
			return nil
		}

		err := os.Remove(fp)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (b *Backlab) getBackupFiles() ([]os.FileInfo, error) {
	return ioutil.ReadDir(*b.BackupPath)
}

func (b *Backlab) newestBackupFile() (*string, error) {
	var latestTimestamp int64 = 0
	var latestBackupPath *string
	b.loopOverBackupFiles(func (f os.FileInfo, fp string, backupTimestamp int64) error {
		if backupTimestamp > latestTimestamp {
			latestTimestamp = backupTimestamp
			latestBackupPath = &fp
		}

		return nil
	})

	return latestBackupPath, nil
}

func (b *Backlab) loopOverBackupFiles(loopAction func(f os.FileInfo, fp string, backupTimestamp int64) error) error {
	files, err := b.getBackupFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		fp := *b.BackupPath + "/" + f.Name()
		if err != nil {
			return err
		}
		if f.IsDir() {
			continue
		}

		backupTimestampString := f.Name()[:10]
		backupTimestamp, err := strconv.ParseInt(backupTimestampString, 10, 64)
		if err != nil {
			return err
		}

		err = loopAction(f, fp, backupTimestamp)
		if err != nil {
			return err
		}
	}

	return nil
}