package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/mediocregopher/radix/v4"
)

var ctx = context.Background()

//Redis acl definitions
var min_acl [4]string = [4]string{"+schema.help", "+schema.execute_query_lua", "+schema.upsert_row", "~*"}
var safe_acl [3]string = [3]string{"+@all", "-@dangerous", "~*"}

func UpsertEntity(w http.ResponseWriter, r *http.Request) {
	var hdr = r.Header["Authorization"]
	if len(hdr) == 0 {
		fmt.Fprintf(w, "Auth failed - no auth header")
		return
	}
	switch r.Method {
	case "DELETE":
		//calculate the key name and delete it
		var urls []string = strings.Split(r.URL.Path, "/")
		if len(urls) < 4 {
			fmt.Fprintf(w, "URL is missing a record id to delete")
			break
		}

		var obj_name = urls[2]
		var id = urls[3]

		params := make([]string, 1)
		key_name := fmt.Sprintf("%s_%s", obj_name, id)
		params[0] = key_name
		var retVal string
		retVal = execute_redis_command(hdr[0], "DEL", params)
		fmt.Fprintf(w, retVal)

	case "POST":
		var obj_name = strings.Split(r.URL.Path, "/")[2]
		fmt.Println(obj_name)
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			break
		}
		var id = r.Form.Get("id")
		var cnt = len(r.Form) * 2
		if id == "" {
			id = "-1"
		}
		params := make([]string, cnt+2)
		params[0] = id
		params[1] = obj_name
		var ii = 2
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
		retVal = execute_redis_command(hdr[0], "schema.upsert_row", params)
		fmt.Fprintf(w, retVal)
	default:
		fmt.Fprintf(w, "Request method %s is not supported", r.Method)
	}
}

func ExecuteScript(w http.ResponseWriter, r *http.Request) {
	var hdr = r.Header["Authorization"]
	if len(hdr) == 0 {
		fmt.Fprintf(w, "Auth failed - no auth header")
		return
	}
	switch r.Method {
	case "POST":
		var obj_name = strings.Split(r.URL.Path, "/")[2]
		fmt.Println(obj_name)
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			break
		}
		var arr_len int = len(r.Form)
		var params = make([]string, arr_len+1)
		params[0] = obj_name
		var ii = 1
		for key := range r.Form { // range over map
			params[ii] = key
			ii++
			fmt.Println(key)
		}
		var retVal string
		retVal = execute_redis_command(hdr[0], "schema.execute_query_lua", params)
		fmt.Fprintf(w, retVal)
	default:
		fmt.Fprintf(w, "Request method %s is not supported", r.Method)
	}
}

func ExecuteAnyCommand(w http.ResponseWriter, r *http.Request) {
	var hdr = r.Header["Authorization"]
	if len(hdr) == 0 {
		fmt.Fprintf(w, "Auth failed - no auth header")
		return
	}
	switch r.Method {
	case "POST":
		var command = strings.Split(r.URL.Path, "/")[2]
		fmt.Println(command)
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			break
		}
		var arr_len int = len(r.Form)
		var params = make([]string, arr_len)

		for key, values := range r.Form { // range over map
			for _, value := range values { // range over []string
				fmt.Println(key, value)
				iKey, err := strconv.Atoi(key)
				if err != nil {
					fmt.Fprintf(w, "key is expected to be a number")
				}
				params[iKey] = value
			}
		}
		var retVal string
		retVal = execute_redis_command(hdr[0], command, params)
		fmt.Fprintf(w, retVal)
	default:
		fmt.Fprintf(w, "Request method %s is not supported", r.Method)
	}
}

func RegisterClient(w http.ResponseWriter, r *http.Request) {
	var rdb = redis_auth(r.Header["Authorization"][0])
	if rdb == nil {
		fmt.Fprintf(w, "Auth failed")
		return
	}
	switch r.Method {
	case "POST":
		if err := r.ParseForm(); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			break
		}
		if len(r.Form.Get("client_name")) == 0 || len(r.Form.Get("client_key")) == 0 {
			fmt.Fprintf(w, "client name or key not supplied")
			break
		}
		var retVal string
		var err error
		if r.Form.Get("client_type") == "safe_acl" {
			err = rdb.Do(ctx, radix.FlatCmd(&retVal, "acl", "setuser", r.Form.Get("client_name"), "on", fmt.Sprintf(">%s", r.Form.Get("client_key")), safe_acl))
		} else {
			err = rdb.Do(ctx, radix.FlatCmd(&retVal, "acl", "setuser", r.Form.Get("client_name"), "on", fmt.Sprintf(">%s", r.Form.Get("client_key")), min_acl))
		}
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(w, retVal)
	default:
		fmt.Fprintf(w, "Request method %s is not supported", r.Method)
	}
	rdb.Close()
}

func Ping(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Pong. Port is: %s", os.Getenv("OS_PORT"))
}

func execute_redis_command(header, command string, params []string) string {
	var rdb = redis_auth(header)
	if rdb == nil {
		return "Can't connect to Redis"
	}
	var retVal interface{}
	err := rdb.Do(ctx, radix.FlatCmd(&retVal, command, params))
	if err != nil {
		panic(err)
	}
	//identify the result type (array vs. individual value)
	ts := fmt.Sprintf("%T", retVal)
	var retString string
	switch ts {
	case "[]interface {}":
		retString = redis_format_results(retVal.([]interface{}))
	case "[]uint8":
		retString = string(retVal.([]byte))
	case "string":
		retString = retVal.(string)
	case "int64":
		retString = fmt.Sprintf("%d", retVal.(int64))
	}
	rdb.Close()
	return retString
}

func redis_format_results(retVal []interface{}) string {

	valArr := make([]string, len(retVal))
	for ii := 0; ii < len(retVal); ii++ {
		valArr[ii] = fmt.Sprintf("%s", retVal[ii])
	}
	return strings.Join(valArr, " ")
}

func redis_auth(header string) radix.Client {
	var credstr = strings.Split(header, " ")[1]
	auth, err := base64.StdEncoding.DecodeString(credstr)
	if err != nil {
		fmt.Println("error:", err)
		return nil
	}
	var creds []string = strings.Split(string(auth), ":")
	return redis_connect(creds[0], creds[1])
}

func redis_connect(user, password string) radix.Client {

	var redis_ip = os.Getenv("REDIS_IP")
	if len(redis_ip) == 0 {
		redis_ip = "127.0.0.1"
	}
	var redis_port = os.Getenv("REDIS_PORT")
	if len(redis_port) == 0 {
		redis_port = "6379"
	}
	redis_connection := fmt.Sprintf("%s:%s", redis_ip, redis_port)
	var d radix.Dialer
	d.AuthPass = password
	d.AuthUser = user
	client, err := d.Dial(ctx, "tcp", redis_connection)
	if err != nil {
		panic(err)
	}

	return client
}

func main() {
	//ping test
	http.HandleFunc("/ping", Ping)
	//upsert to an entity
	http.HandleFunc("/e/", UpsertEntity)
	//execute a lua script
	http.HandleFunc("/s/", ExecuteScript)
	//register client
	http.HandleFunc("/register", RegisterClient)
	//execute any Redis command
	http.HandleFunc("/command/", ExecuteAnyCommand)

	var port_string = os.Getenv("OS_PORT")
	if len(port_string) == 0 {
		port_string = ":80"
	} else {
		port_string = fmt.Sprintf(":%s", port_string)
	}
	http.ListenAndServe(port_string, nil)
}
