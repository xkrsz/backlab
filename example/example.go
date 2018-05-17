package main

import (
	"os"
	"github.com/krszwsk/backlab"
)

func main() {
	bl, _ := backlab.Init(backlab.Config{
		Credentials: backlab.Credentials{
			AccountID: os.Getenv("B2_ACCOUNT_ID"),
			ApplicationKey: os.Getenv("B2_APPLICATION_KEY"),
		},
		BucketName: "gitlab-backups",
	})

	bl.Backup("./test.zip")
}
