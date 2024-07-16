package pkg

import (
	"bytes"
	"database/sql"
	"embed"
	"log"
	"path"

	_ "modernc.org/sqlite"
)

var LogDB *sql.DB

func newDeployLogger(repoName string) (*log.Logger, chan bool) {
	b := bytes.Buffer{}
	ready := make(chan bool)
	go func() {
		<-ready
		_, err := LogDB.Exec(`INSERT INTO logs (repo, time, data)
			VALUES ($1, unixepoch('now'), $2)`, repoName, b.String())
		if err != nil {
			log.Print(err)
		}
	}()
	return log.New(&b, "", log.LstdFlags), ready
}

//go:embed sql/*.sql
var dbMigrations embed.FS

func migrateDB() error {
	_, err := LogDB.Exec("CREATE TABLE IF NOT EXISTS migrations (name text)")
	if err != nil {
		return err
	}
	LogDB.Exec("BEGIN")
	defer LogDB.Exec("ROLLBACK")
	files, err := dbMigrations.ReadDir("sql")
	if err != nil {
		return err
	}
	for _, v := range files {
		if !v.IsDir() {
			var name string
			err := LogDB.QueryRow("SELECT name FROM migrations WHERE name = $1",
				v.Name()).Scan(&name)
			if err != nil && err != sql.ErrNoRows {
				return err
			}

			if err == sql.ErrNoRows {
				content, err := dbMigrations.ReadFile(path.Join("sql", v.Name()))
				if err != nil {
					return err
				}
				log.Printf("Applying migration: %s", v.Name())
				if _, err = LogDB.Exec(string(content)); err != nil {
					return err
				}
				if _, err = LogDB.Exec("INSERT INTO migrations VALUES ($1)",
					v.Name()); err != nil {
					return err
				}
			}
		}
	}
	LogDB.Exec("COMMIT")
	return nil
}

func InitDB(name string) error {
	var err error
	log.Print("Initializing log database")
	LogDB, err = sql.Open("sqlite", name)
	if err != nil {
		return err
	}
	return migrateDB()
}
