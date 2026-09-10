package main

import (
	"database/sql"
	"log"

	db "github.com/Teixeiraass/ground_guard_be/db/sqlc"
	"github.com/Teixeiraass/ground_guard_be/internal/handler"
	"github.com/Teixeiraass/ground_guard_be/mqtt/client"
	"github.com/Teixeiraass/ground_guard_be/util"

	"time"
	_ "time/tzdata"

	_ "github.com/lib/pq"

	_ "github.com/Teixeiraass/ground_guard_be/docs"
)

func init() {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err == nil {
		time.Local = loc
	}
}

// @title           Ground Guard API
// @version         1.0.0
// @description     API REST do Ground Guard, uma plataforma IoT para monitoramento, automação e operação de jardins e plantas.
// @description     O backend centraliza autenticação, vínculo de dispositivos, preferências e histórico de irrigação, comandos remotos, telemetria, conteúdos de suporte e relatórios operacionais.
// @description     O projeto foi estruturado como produto comercial: pensado para onboarding rápido de novos desenvolvedores, expansão por módulos e manutenção contínua em ambiente de produção.
// @termsOfService  https://groundguard.com/terms
// @contact.name    Guilherme Teixeira
// @contact.email   contato@groundguard.com
// @license.name    MIT
// @host            api-dev.ground-guard.com.br
// @BasePath        /api/v1
// @schemes         https
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

	mqttClient, err := client.NewPahoClient(config)
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
