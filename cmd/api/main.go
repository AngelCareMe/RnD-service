package main

import (
	"RnD-service/internal/adapter/cbr"
	"RnD-service/internal/adapter/postgres"
	"RnD-service/internal/handler"
	"RnD-service/internal/service"
	"RnD-service/internal/usecase"
	"RnD-service/pkg/config"
	"RnD-service/pkg/logger"
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log := logger.Init(cfg.Log.Level)

	log.Info("Starting app...")

	// initialize db pools
	dbPool, err := postgres.InitDBPool(*cfg, log)
	if err != nil {
		log.Fatalf("Failed to initialize db pools")
	}

	// initialize adapters
	cbr := cbr.NewClient(log)
	log.Info("Initialized API")

	db := postgres.NewPostgresRepo(dbPool, log)
	log.Info("Initialized database pool")

	// initialize service
	currencyService := service.NewRateService(cbr, *db, log)
	log.Info("Initialized service layer")

	// initialize usecase
	currencyUsecase := usecase.NewCurrencyUsecase(*currencyService, log)
	log.Info("Initialized usecase layer")

	currencyHandler := handler.NewRateHandler(*currencyUsecase, log)

	r := gin.Default()

	r.GET("/currency/rates", currencyHandler.StoreRatesFromCBR) // Загрузка курсов с ЦБ
	r.GET("/currency/get", currencyHandler.GetRateByCharCode)   // Получение курса по коду

	log.Info("Server starting on port 8080...")
	r.Run(":8080")
}
