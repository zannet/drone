package migration

//go:generate go-bindata-assetfs -pkg migration -o migration_gen.go sqlite3/ mysql/ postgres/
