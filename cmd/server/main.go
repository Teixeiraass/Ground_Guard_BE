package main

import (
	"database/sql"
	"log"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/handler"
	"github.com/Teixeiraass/ground_guard_be/mqtt"
	"github.com/Teixeiraass/ground_guard_be/util"

	_ "github.com/lib/pq"

	_ "github.com/Teixeiraass/ground_guard_be/docs"
)

// @title           Ground Guard API
// @version         1.0.0
// @description     API REST do Ground Guard, uma plataforma IoT para monitoramento e automação de jardins e plantas.
// @description     Permite gerenciamento de dispositivos, preferências de irrigação, monitoramento ambiental e acionamento remoto de irrigação.
// @description     Desenvolvido como TCC e preparado para evolução comercial.
// @termsOfService  https://groundguard.com/terms
// @contact.name    Guilherme Teixeira
// @contact.email   contato@groundguard.com
// @license.name    MIT
// @host            localhost:8080
// @BasePath        /api/v1
// @schemes         http https
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
// @description     Informe: Bearer {token}
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

	mqttClient, err := mqtt.NewPahoClient(config)
	if err != nil {
		log.Fatal("cannot connect to mqtt broker:", err)
	}
	defer mqttClient.Close()

	server, err := handler.NewServer(config, store, mqttClient)
	if err != nil {
		log.Fatal("cannot create server: ", err)
	}

	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
