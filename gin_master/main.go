package main

// @title Swagger Example API
// @version 1.0
// @description This is a sample server.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api

import "github.com/zzu-andrew/go-example/gin_master/api"

//go:generate swag init --parseDependency --parseDepth=6 --instanceName admin -o ./doc/admin
func main() {
	api.Router()
}
