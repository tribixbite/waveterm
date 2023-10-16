# SH2 Server Commands

```bash
# @scripthaus command dump-schema-dev
sqlite3 /Users/mike/prompt-dev/prompt.db .schema > db/schema.sql
```

```bash
# @scripthaus command opendb-dev
sqlite3 /Users/mike/prompt-dev/prompt.db
```

```bash
# @scripthaus command build
go build -ldflags "-X main.BuildTime=$(date +'%Y%m%d%H%M')" -o bin/local-server ./cmd
```