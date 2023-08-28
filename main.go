package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var rnd *renderer.Render
var db *mgo.Database

const (
	hostName       string = "localhost:27017"
	port           string = "9000"
	dbName         string = "demo_todo"
	collectionName string = "todo"
)

type (
	todo struct {
		ID        string    `json:"id"`
		CreatedAt time.Time `json:"createdAt"`
		Title     string    `json:"title"`
		Completed bool      `json:"completed"`
	}
	todoModel struct {
		ID        bson.ObjectId `bson:"_id, ommitempty"`
		CreatedAt time.Time     `bson:"createdAt"`
		Title     string        `bson:"title"`
		Completed bool          `bson:"comoleted"`
	}
)

func init() {
	rnd = renderer.New()
	sess, err := mgo.Dial(hostName)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
	db := sess.DB(dbName)
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := rnd.Render(w, http.StatusOK, []string{"static/home.tpl"})
	checkErr(err)
}

func main() {

	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/todo", todoHandler())

	srv := http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatal("Listen error: %s \n", err)
		}
	}()

	<-stopChan
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
	log.Println("server shutting down gracefully")

}

func todoHandler() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", fetchTodos)
		r.Put("/", updateTodo)
		r.Post("/", addTodo)
		r.Delete("/", deleteTodo)

	})
	return rg
}

func fetchTodos(w http.ResponseWriter, r *http.Request) {
	todos := []todoModel{}
	err := db.C(collectionName).Find(bson.M{}).All(&todos)
	if err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "failed to fetch data",
			"err":     err,
		})
		return
	}
	todoList := []todo{}

	for _, t := range todos {
		todoList = append(todoList, todo{
			ID:        t.ID.Hex(),
			Title:     t.Title,
			Completed: t.Completed,
			CreatedAt: t.CreatedAt,
		})
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todoList,
	})
}
