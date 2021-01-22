package function

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
)

type AddTodoReq struct {
	Key  string `json:"key"`
	Todo string `json:"todo"`
}

type GetTodoRes struct {
	Todo string `json:"todo"`
}

type GetTodoReq struct {
	Key string `json:"key"`
}

func Handle(w http.ResponseWriter, r *http.Request) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: "",
		DB:       0,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal(err)
	}

	if r.Method == http.MethodPost && r.URL.Path == "/add" {
		todo := AddTodoReq{}
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(body, &todo); err != nil {
			http.Error(w, fmt.Sprint("unable to unmarshall request body"), http.StatusBadRequest)
		}

		if err := rdb.Set(context.Background(), todo.Key, todo.Todo, time.Hour).Err(); err != nil {
			http.Error(w, fmt.Sprintf("unable to insert todo: %v", err), http.StatusInternalServerError)
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Todo added"))

	} else if r.Method == http.MethodPost && r.URL.Path == "/get" {
		reqBody := GetTodoReq{}
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(body, &reqBody); err != nil {
			http.Error(w, fmt.Sprint("unable to unmarshall request body"), http.StatusBadRequest)
		}
		todo, err := rdb.Get(context.Background(), reqBody.Key).Result()
		if err == redis.Nil {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("no value found"))
			return
		}
		if err != nil {
			http.Error(w, fmt.Sprint(err), http.StatusInternalServerError)
			return
		}
		todoRes := GetTodoRes{
			Todo: todo,
		}
		res, err := json.Marshal(todoRes)
		if err != nil {
			http.Error(w, fmt.Sprintf("unale marshal: %s", err.Error()), http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		w.Write(res)
	}
}
