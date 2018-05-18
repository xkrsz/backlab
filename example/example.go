package main

import (
	"github.com/krszwsk/backlab"
	"os"
)

func main() {
	bl, _ := backlab.New(backlab.Config{
		Credentials: backlab.Credentials{
			AccountID: os.Getenv("B2_ACCOUNT_ID"),
			ApplicationKey: os.Getenv("B2_APPLICATION_KEY"),
		},
		BucketName: "gitlab-backups",
	})

	bl.Backup("./test.zip")
}
