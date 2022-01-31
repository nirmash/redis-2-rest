package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mediocregopher/radix/v4"
)

var ctx = context.Background()
var rdb = RedisConnect()

func UpsertEntity(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprintf(w, "This is GET request at path = %s", r.URL.Path)
	case "DELETE":
		//calculate the key name and delete it
		var obj_name = strings.Split(r.URL.Path, "/")[2]
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		var id = r.Form.Get("id")
		if id == "" {
			fmt.Fprintf(w, "Record Id not specified")
			return
		}
		key_name := fmt.Sprintf("%s_%s", obj_name, id)
		var retVal string
		err := rdb.Do(ctx, radix.FlatCmd(&retVal, "DEL", key_name))
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, retVal)
		return

	case "POST":
		var obj_name = strings.Split(r.URL.Path, "/")[2]
		fmt.Println(obj_name)
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		var id = r.Form.Get("id")
		var cnt = len(r.Form) * 2
		if id == "" {
			id = "-1"
		} else {
			cnt = cnt - 2
		}
		params := make([]string, cnt)
		var ii = 0
		for key, values := range r.Form { // range over map
			for _, value := range values { // range over []string
				fmt.Println(key, value)
				if key != "id" {
					params[ii] = key
					ii++
					params[ii] = value
					ii++
				}
			}
		}
		var retVal string
		err := rdb.Do(ctx, radix.FlatCmd(&retVal, "schema.upsert_row", id, obj_name, params))
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, retVal)
		return
	default:
		fmt.Fprintf(w, "Request method %s is not supported", r.Method)
	}
}

func ExecuteScript(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		fmt.Fprintf(w, "This is GET request at path = %s", r.URL.Path)
	case "POST":
		var obj_name = strings.Split(r.URL.Path, "/")[2]
		fmt.Println(obj_name)
		//connection impersonation to come later
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		var arr_len int = len(r.Form)
		var params = make([]string, arr_len)
		var ii = 0
		for key := range r.Form { // range over map
			params[ii] = key
			ii++
			fmt.Println(key)
		}
		var retVal []interface{}
		err := rdb.Do(ctx, radix.FlatCmd(&retVal, "schema.execute_query_lua", obj_name, params))
		if err != nil {
			panic(err)
		}
		var str = FormatRedisResult(retVal)
		fmt.Fprintf(w, str)
	default:
		fmt.Fprintf(w, "Request method %s is not supported", r.Method)
	}
}

func FormatRedisResult(retVal []interface{}) string {
	valArr := make([]string, len(retVal))
	for ii := 0; ii < len(retVal); ii++ {
		valArr[ii] = fmt.Sprintf("%s", retVal[ii])
	}
	return strings.Join(valArr, "\r")
}

func RedisConnect() radix.Client {
	client, err := (radix.PoolConfig{}).New(ctx, "tcp", "127.0.0.1:6379")
	if err != nil {
		panic(err)
	}
	return client
}

func main() {
	//upsert to an entity
	http.HandleFunc("/e/", UpsertEntity)
	//execute a lua script
	http.HandleFunc("/s/", ExecuteScript)

	fmt.Println("Listening on port 5050...")

	http.ListenAndServe(":5050", nil)
}
