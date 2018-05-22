# gitlab-backblaze-backup

[![GoDoc](https://godoc.org/github.com/krszwsk/backlab?status.svg)](https://godoc.org/gopkg.in/krszwsk/backlab.v0)

## Usage
```go
import "gopkg.in/krszwsk/backlab.v0"
```

```go
bl := backlab.New(backlab.Config{
  Credentials: backlab.Credentials{
    AccountID: os.Getenv("B2_ACCOUNT_ID"),
    ApplicationKey: os.Getenv("B2_APPLICATION_KEY"),
  },
  BucketName: "backlab-gitlab-backups",
  PreserveFor: 60 * 60 * 24 * 7, // 7 days
  BackupPath: "/var/opt/gitlab/backups",
})
```

### Perform a backup, including removing old local and remote backups
```go
bl.Backup()
```
