package function

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var schema = `
CREATE TABLE IF NOT EXISTS todo (
	id SERIAL PRIMARY KEY,
	todo TEXT,
	author TEXT NOT NULL
);`

var db *sqlx.DB

type Todo struct {
	ID     string `db:"id" json:"id"`
	Todo   string `db:"todo" json:"todo"`
	Author string `db:"author" json:"author"`
}

func list() ([]Todo, error) {
	rows, err := db.Query("SELECT id, todo, author FROM todo;")
	if err != nil {
		return nil, err
	}

	todos := []Todo{}
	defer rows.Close()
	for rows.Next() {
		result := Todo{}
		scanErr := rows.Scan(&result.ID, &result.Todo, &result.Author)
		if scanErr != nil {
			log.Printf("scan err: %v", scanErr)
		}
		todos = append(todos, result)
	}
	return todos, nil
}

func insert(todo Todo) error {
	_, err := db.Exec("INSERT INTO todo (todo, author) VALUES ($1, $2);", todo.Todo, todo.Author)
	return err
}

func Handle(w http.ResponseWriter, r *http.Request) {
	password, _ := ioutil.ReadFile("/var/openfaas/secrets/postgres-passwd")
	databaseURL := fmt.Sprintf("host=%s port=%s user=postgres password=%s dbname=postgres sslmode=disable", os.Getenv("HOST"), os.Getenv("PORT"), password)
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		log.Fatalln(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err.Error())
	}
	db.MustExec(schema)

	if r.Method == http.MethodPost && r.URL.Path == "/add" {
		todo := Todo{}
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(body, &todo); err != nil {
			http.Error(w, fmt.Sprint("unable to unmarshall request body"), http.StatusBadRequest)
		}

		if _, err := db.Exec("INSERT INTO todo (todo, author) VALUES ($1, $2);", todo.Todo, todo.Author); err != nil {
			http.Error(w, fmt.Sprintf("unable to insert todo: %s", err.Error()), http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Todo added"))

	} else if r.Method == http.MethodGet && r.URL.Path == "/list" {
		rows, err := db.Query("SELECT id, todo, author FROM todo;")
		if err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
		}

		todos := []Todo{}
		defer rows.Close()
		for rows.Next() {
			result := Todo{}
			scanErr := rows.Scan(&result.ID, &result.Todo, &result.Author)
			if scanErr != nil {
				log.Printf("scan err: %v", scanErr)
			}
			todos = append(todos, result)
		}

		res, err := json.Marshal(todos)
		if err != nil {
			http.Error(w, fmt.Sprintf("unale marshal: %s", err.Error()), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}
