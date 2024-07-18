package pkg

import (
	"bytes"
	"database/sql"
	"embed"
	"log"
	"path"
	"time"

	_ "modernc.org/sqlite"
)

var LogDB *sql.DB

func newDeployLogger(repoName string) (*log.Logger, chan bool) {
	b := bytes.Buffer{}
	ready := make(chan bool)
	go func() {
		s := <-ready
		_, err := LogDB.Exec(`INSERT INTO logs (repo, time, data, success)
			VALUES ($1, unixepoch('now'), $2, $3)`, repoName, b.String(), s)
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

type logRecord struct {
	Id      int
	Name    string
	Time    time.Time
	Data    string
	Success bool
}

func getLogs(name string) []logRecord {
	rows, err := LogDB.Query("SELECT id, time, success FROM logs WHERE repo = $1 ORDER BY ID DESC", name)
	if err != nil {
		log.Print(err)
		return []logRecord{}
	}
	var out []logRecord
	for rows.Next() {
		var id, t int
		var s bool
		rows.Scan(&id, &t, &s)
		out = append(out, logRecord{
			Id:      id,
			Time:    time.Unix(int64(t), 0),
			Success: s,
		})
	}
	return out
}

func getLog(id int) logRecord {
	l := logRecord{}
	var t int64
	err := LogDB.QueryRow("SELECT id, repo, time, data FROM logs WHERE id = $1", id).
		Scan(&l.Id, &l.Name, &t, &l.Data)
	if err != nil {
		log.Print(err)
		return logRecord{Id: -1}
	}
	l.Time = time.Unix(t, 0)
	return l
}
