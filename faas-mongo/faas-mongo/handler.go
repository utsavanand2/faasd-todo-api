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

	"go.mongodb.org/mongo-driver/bson/primitive"

	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Todo struct {
	ID     primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Todo   string             `json:"todo" bson:"todo"`
	Author string             `json:"author" bson:"author"`
}

type TodoAddRes struct {
	InsertID primitive.ObjectID `json:"id"`
}

type TodoDeleteRes struct {
	DeleteCount string `json:"deletedTodos"`
}

type TodoUpdateRes struct {
	UpdateCount string `json:"updatedTodos"`
}

func Handle(w http.ResponseWriter, r *http.Request) {
	username, _ := ioutil.ReadFile("/var/openfaas/secrets/mongo-user")
	password, _ := ioutil.ReadFile("/var/openfaas/secrets/mongo-passwd")
	creds := options.Credential{
		Username: string(username),
		Password: string(password),
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGO_URI")).SetAuth(creds))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Disconnect(context.Background()); err != nil {
			panic(err)
		}
	}()

	collection := client.Database("test").Collection("todo")

	if r.Method == http.MethodPost && r.URL.Path == "/add" {
		reqAddTodo := Todo{}
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(body, &reqAddTodo); err != nil {
			http.Error(w, "Bad request body", http.StatusBadRequest)
			log.Printf("err unmarshalling req: %v", err)
			return
		}
		insertResult, err := collection.InsertOne(context.Background(), reqAddTodo)
		if err != nil {
			http.Error(w, "Err Inserting Document", http.StatusInternalServerError)
			log.Printf("err inserting doc: %v", err)
			return
		}
		id, _ := insertResult.InsertedID.(primitive.ObjectID)
		res, err := json.Marshal(TodoAddRes{
			InsertID: id,
		})
		if err != nil {
			http.Error(w, "Err marshalling response", http.StatusInternalServerError)
			log.Printf("err marshalling res: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	} else if r.Method == http.MethodGet && r.URL.Path == "/get" {
		todos := []Todo{}
		author := r.FormValue("author")
		log.Printf("form value of author: %s", author)
		filter := bson.M{"author": bson.D{
			{"$regex", primitive.Regex{Pattern: author, Options: "i"}},
		}}
		cursor, err := collection.Find(context.Background(), filter)
		if err != nil {
			http.Error(w, "Err fetching todos", http.StatusInternalServerError)
			log.Printf("err fetching docs: %v", err)
			return
		}
		defer cursor.Close(context.Background())
		if err = cursor.All(context.Background(), &todos); err != nil {
			http.Error(w, "Err unmarshalling todos from cursor to struct", http.StatusInternalServerError)
			log.Printf("err unmarshalling todos from cursor to struct: %v", err)
			return
		}
		res, err := json.Marshal(todos)
		if err != nil {
			http.Error(w, "Err marshalling response", http.StatusInternalServerError)
			log.Printf("err marshalling res: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)

	} else if r.Method == http.MethodGet && r.URL.Path == "/todos" {
		todos := []Todo{}
		cursor, err := collection.Find(context.Background(), bson.D{})
		if err != nil {
			http.Error(w, "Err fetching todos", http.StatusInternalServerError)
			log.Printf("err fetching docs: %v", err)
			return
		}
		defer cursor.Close(context.Background())
		if err = cursor.All(context.Background(), &todos); err != nil {
			http.Error(w, "Err unmarshalling todos from cursor to struct", http.StatusInternalServerError)
			log.Printf("err unmarshalling todos from cursor to struct: %v", err)
			return
		}
		res, err := json.Marshal(todos)
		if err != nil {
			http.Error(w, "Err marshalling response", http.StatusInternalServerError)
			log.Printf("err marshalling res: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	} else if r.Method == http.MethodDelete && r.URL.Path == "/delete" {
		id := r.FormValue("id")
		log.Printf("form value of id: %s", id)
		idPrimitive, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			http.Error(w, "Invalid todo ID", http.StatusBadRequest)
			log.Printf("err converting id to primitive objectID: %v", err)
			return
		}
		filter := bson.M{"_id": idPrimitive}
		result, err := collection.DeleteOne(context.Background(), filter)
		if err != nil {
			http.Error(w, "Err deleting todo", http.StatusInternalServerError)
			log.Printf("err deleting document: %v", err)
			return
		}

		res, err := json.Marshal(TodoDeleteRes{
			DeleteCount: fmt.Sprint(result.DeletedCount),
		})
		if err != nil {
			http.Error(w, "Err marshalling response", http.StatusInternalServerError)
			log.Printf("err marshalling res: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	} else if r.Method == http.MethodPut && r.URL.Path == "/update" {
		reqUpdateTodo := Todo{}
		defer r.Body.Close()
		body, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(body, &reqUpdateTodo); err != nil {
			http.Error(w, "Bad request body", http.StatusBadRequest)
			log.Printf("err unmarshalling req: %v", err)
			return
		}

		filter := bson.M{"_id": reqUpdateTodo.ID}

		newTodo := bson.D{{"$set", Todo{
			Todo:   reqUpdateTodo.Todo,
			Author: reqUpdateTodo.Author,
		}}}

		result, err := collection.UpdateOne(context.Background(), filter, newTodo)
		if err != nil {
			http.Error(w, "Err updating todo", http.StatusInternalServerError)
			log.Printf("err updating document: %v", err)
			return
		}

		res, err := json.Marshal(TodoUpdateRes{
			UpdateCount: fmt.Sprint(result.ModifiedCount),
		})
		if err != nil {
			http.Error(w, "Err marshalling response", http.StatusInternalServerError)
			log.Printf("err marshalling res: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	}
}
