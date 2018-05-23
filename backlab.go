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

type BackblazeFile struct {
	Name string
	ID string
}

// Credentials is a type alias for backblaze.Credentials.
type Credentials backblaze.Credentials

// Config is a parameter used in Init for configuring backlab.
type Config struct {
	Credentials
	BucketName string
	// PreserveFor defines how long should the backups be kept in a bucket and locally. Doesn't delete any by default.
	PreserveFor int64
	BackupPath string
}

// Backlab is a main struct.
type Backlab struct {
	Config

	b2 *backblaze.B2
	bucket *backblaze.Bucket
	backupExpiryTimestamp int64
}

// New configures connection with Backblaze and saves configuration to Backlab instance.
func New(config Config) *Backlab {
	b := &Backlab{
		Config: config,
	}

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID:      b.AccountID,
		ApplicationKey: b.ApplicationKey,
	})
	must(err)

	b.b2 = b2

	var bucket *backblaze.Bucket
	if &b.BucketName == nil {
		randomString := uniuri.New()
		bucket, err = b.b2.CreateBucket("backlab-gitlab-backups-"+randomString, backblaze.AllPrivate)
	} else {
		bucket, err = b.b2.Bucket(b.BucketName)
	}
	must(err)

	b.bucket = bucket

	b.backupExpiryTimestamp = time.Now().Unix() - b.PreserveFor

	return b
}

// Backup creates a new backup, removes old backups, and uploads the new backup to Backblaze
func (b *Backlab) Backup() error {
	err := b.CreateBackup()
	if err != nil {
		return err
	}
	err = b.RemoveOldLocalBackups()
	if err != nil {
		return err
	}
	archivePath, err := b.newestBackupFile()
	if err != nil {
		return err
	}
	err = b.UploadBackup(archivePath)
	if err != nil {
		return err
	}
	err = b.RemoveOldRemoteBackups()
	if err != nil {
		return err
	}

	return nil
}

// BackupArchive backups a file to Backblaze.
func (b *Backlab) UploadBackup(archivePath string) error {

	reader, _ := os.Open(archivePath)
	name := filepath.Base(archivePath)
	metadata := make(map[string]string)

	_, err := b.bucket.UploadFile(name, metadata, reader)
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
	return err
}

// RemoveOldRemoteBackups removes old backups from Backblaze, based on Config.PreserveFor value.
func (b *Backlab) RemoveOldRemoteBackups() error {
	result, err := b.bucket.ListFileVersions("", "", 100)
	if err != nil {
		return err
	}

	var removedFiles []BackblazeFile

	for _, f := range result.Files {
		backupTimestamp, err := b.extractTimestampFromFilename(f.Name)
		if err != nil {
			return err
		}

		if *backupTimestamp > b.backupExpiryTimestamp {
			continue
		}

		_, err = b.bucket.DeleteFileVersion(f.Name, f.ID)
		if err != nil {
			return err
		}

		removedFiles = append(removedFiles, BackblazeFile{
			f.Name,
			f.ID,
		})
	}

	return nil
}

func (b *Backlab) extractTimestampFromFilename(filename string) (*int64, error) {
	backupTimestampString := filename[:10]
	backupTimestamp, err := strconv.ParseInt(backupTimestampString, 10, 64)
	if err != nil {
		return nil, err
	}

	return &backupTimestamp, nil
}

func (b *Backlab) getBackupFiles() ([]os.FileInfo, error) {
	return ioutil.ReadDir(b.BackupPath)
}

func (b *Backlab) newestBackupFile() (string, error) {
	var latestTimestamp int64 = 0
	var latestBackupPath *string
	err := b.loopOverBackupFiles(func (f os.FileInfo, fp string, backupTimestamp int64) error {
		if backupTimestamp > latestTimestamp {
			latestTimestamp = backupTimestamp
			latestBackupPath = &fp
		}

		return nil
	})

	return *latestBackupPath, err
}

func (b *Backlab) loopOverBackupFiles(loopAction func(f os.FileInfo, fp string, backupTimestamp int64) error) error {
	files, err := b.getBackupFiles()
	if err != nil {
		return err
	}
	for _, f := range files {
		fp := b.BackupPath + "/" + f.Name()
		if err != nil {
			return err
		}
		if f.IsDir() {
			continue
		}

		backupTimestamp, err := b.extractTimestampFromFilename(f.Name())
		if err != nil {
			return err
		}


		err = loopAction(f, fp, *backupTimestamp)
		if err != nil {
			return err
		}
	}

	return nil
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
