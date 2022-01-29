package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

func MainHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprintf(w, "This is GET request at path = %s", r.URL.Path)
	case "POST":
		var obj_name = strings.Trim(r.URL.Path, "/")
		fmt.Println(obj_name)
		//connection impersonation to come later
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		var id = r.Form.Get("id")
		if id == "" {
			id = "-1"
		}
		for key, values := range r.Form { // range over map
			for _, value := range values { // range over []string
				fmt.Println(key, value)
				if key != "id" {
					id = UpsertRecord(RedisConnect(), id, obj_name, key, value)
				}
			}
		}
		fmt.Println(w, "HTTP/1.1 200 OK\r")
	default:
		fmt.Fprintf(w, "Request method %s is not supported", r.Method)
	}
}

func UpsertRecord(rdb *redis.Client, id string, table string, key string, value string) string {
	val, err := rdb.Do(ctx, "schema.upsert_row", id, table, key, value).Result()
	if err != nil {
		return "fail"
	}
	rdb.Close()
	return val.(string)
}

func RedisConnect() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	return rdb
}

func main() {
	http.HandleFunc("/", MainHandler)

	fmt.Println("Listening on port 5050...")

	http.ListenAndServe(":5050", nil)
}
