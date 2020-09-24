# Auth service

This is simple auth service with the ability to connect multiple loggers.

The project includes:
* [auth_service](/auth_service) - implements user registration, issuance and updating JWT tokens, storing user information in the database.
* [auth_client](/auth_client) - provides rest api for registration, login, info and JWT tokens update.
* [logger_client](/logger_client) - connects using gRPC to the auth_service and displays logs in a console. 

### How to run?

Check you have installed:
* [Docker](https://docs.docker.com/get-docker/) and [docker compose](https://docs.docker.com/compose/install/)

1.  Go to the project folder
1.  Run `docker-compose build`
1.  Run `docker-compose up`

### Built With

* [Golang](https://golang.org/) - A open source programming language that makes it easy to build simple, reliable, and efficient software
* [gRPC](https://grpc.io/) - A high-performance, open source universal RPC framework
* [JWT](https://jwt.io/) - JSON Web Tokens are an open, industry standard RFC 7519 method for representing claims securely between two parties