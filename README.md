# Redis to REST, securely 
You know I think [Redis](https://redis.io/) is awesome for just about everything. Developers use Redis to accelerate their applications by caching data stored in a durable database or to act as a state layer for distributed components and Microservices. The Redis API is easy-to-use which makes it even more popular with developers, it supports key-value pairs, hashes and all sorts of lists (pun intended). Compared to a relational database, Redis is missing the ability to validate data structures (see my [blog](https://medium.com/@nirmashkowski/how-to-make-redis-play-nice-with-your-data-32b416a4fd05) on the topic). This article looks at a couple of other areas that need to be tackled when using Redis for a distributed application:
- **REST API mapping** - In many cases, it would be much easier to interact with Redis using HTTP endpoints that follows [REST](https://en.wikipedia.org/wiki/Representational_state_transfer) conventions.
- **Security** - If we choose to expose our Redis database through HTTP, we would want to follow the [Principle of Least Privilege](https://en.wikipedia.org/wiki/Principle_of_least_privilege) and allow the client interacting with Redis access to the appropriate subset of Redis command and keys.

This sample in this article can be used with a "vanilla" Redis instance (v6.0+) and with a Redis instance that leverages the `schema` module (source code available [here](https://github.com/nirmash/redis-schema)).

## Building the REST API
In this article, I chose to use [golang](https://go.dev/) (for no particular reason...) to implement an HTTP API that will interact with Redis. The API uses [Radix](https://pkg.go.dev/github.com/mediocregopher/radix/v4@v4.0.0) Redis client and exposes 6 public methods:
- **Ping ("/ping")** - Returns the port number the API is listening on.
- **ExecuteAnyCommand ("/command/")** - Executes any Redis command allowed for the API user. Command name is the url segment after `/command/` and command parameters are submitted as key-value pairs in the POST body where the key name is expected to be the sort order number of the parameter sent to Redis.
- **RegisterClient ("register")** - This command takes a `client_name`, `client_key` and `client_type` key-value pair in the POST body. Client name and key are the user name and password for an API client and `client_type` can be either `safe_acl` or `min_acl` (explained later). Upon success, a new Redis user is created with the password and ACL type provided.

The below commands are dependant on the [schema](https://github.com/nirmash/redis-schema) Redis module that handles Redis data schema validation and allows registration and execution of [Lua scripts](https://www.ibm.com/cloud/blog/a-quick-guide-to-redis-lua-scripting).

- **UpsertEntity ("/e/")** - Takes an entity name and named parameters for it in the POST body to add or update and entity. It also takes an entity name and record id in the url parameters for the DELETE verb. 
- **ExecuteScript ("/s/")** - Takes a lua script name and parameters in the POST body. It is the same as calling `schema.execute_query_lua` command (explained [here](https://github.com/nirmash/redis-schema)).

## Authentication and authorization
The sample in this article uses [basic auth](https://en.wikipedia.org/wiki/Basic_access_authentication) HTTP headers to pass user names and passwords to Redis. Redis 6.0+ uses an [ACL](https://redis.io/topics/acl) subsystem to assign permissions to an individual user which is then used to connect to Redis. When calling the `/register` API in the sample, the call needs to have access to the `ACL SETUSER` command. Redis automatically comes with an admin user called `default` that has access to all Redis commands. 

**Note:** The `default` user comes with an empty password 

## Pre-requisites
Using this sample requires the following.
1. Install [Git](https://git-scm.com/downloads)
2. Install [Docker](https://docs.docker.com/get-docker/)
3. Install [Docker Compose](https://docs.docker.com/compose/install/)


``` bash 
curl --user "tommy:lee" -d "0=MyList" -X  POST "http://localhost/command/smembers"
```