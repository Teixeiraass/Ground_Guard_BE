package main

import (
	"database/sql"
	"log"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/handler"
	"github.com/Teixeiraass/ground_guard_be/util"

	_ "github.com/lib/pq"

	_ "github.com/Teixeiraass/ground_guard_be/docs"
)

// @title			Ground Guard API
// @version		1.0
// @description	API do projeto ground guard automação de jardim 
// @host			localhost:8080
// @BasePath		/
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and your PASETO token.
func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	store := db.NewStore(conn)
	server, err := handler.NewServer(config, store)
	if err != nil {
		log.Fatal("cannot create server: ", err)
	}

	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
