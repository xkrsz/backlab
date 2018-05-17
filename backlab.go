package backlab

import (
	"gopkg.in/kothar/go-backblaze.v0"
	"os"
	"path/filepath"
	"fmt"
)

type Credentials backblaze.Credentials

type Config struct {
	Credentials
	BucketName string
}

type Backlab struct {
	Config

	b2 *backblaze.B2
}

func Init(config Config) (*Backlab, error) {
	b := &Backlab{
		Config: config,
	}

	b2, err := backblaze.NewB2(backblaze.Credentials{
		AccountID: b.AccountID,
		ApplicationKey: b.ApplicationKey,
	})
	if err != nil { panic(err) }

	b.b2 = b2

	return b, nil
}

func (b *Backlab) Backup(archivePath string) {
	var bucket *backblaze.Bucket
	if &b.BucketName == nil {
		bucket, _ = b.b2.CreateBucket("gitlab-backups", backblaze.AllPrivate)
	} else {
		bucket, _ = b.b2.Bucket(b.BucketName)
	}

	reader, _ := os.Open(archivePath)
	name := filepath.Base(archivePath)
	metadata := make(map[string]string)

	file, _ := bucket.UploadFile(name, metadata, reader)
	fmt.Println(file)
}
