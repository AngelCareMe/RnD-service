package main

import (
	"RnD-service/internal/adapter/cbr"
	"RnD-service/internal/adapter/postgres"
	"RnD-service/internal/handler"
	"RnD-service/internal/service"
	"RnD-service/internal/usecase"
	"RnD-service/pkg/config"
	"RnD-service/pkg/logger"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
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
	cbrClient := cbr.NewClient(log)
	log.Info("Initialized API")

	db := postgres.NewPostgresRepo(dbPool, log)
	log.Info("Initialized database pool")

	// initialize service
	currencyService := service.NewRateService(cbrClient, db, log)
	log.Info("Initialized service layer")

	// initialize usecase
	currencyUsecase := usecase.NewCurrencyUsecase(currencyService, log)
	log.Info("Initialized usecase layer")

	currencyHandler := handler.NewRateHandler(currencyUsecase, log)

	r := gin.Default()

	// cors middleware
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:8080", "http://127.0.0.1:8080"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// dashboard usage
	r.Static("/static", "./static")

	// index context
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	r.GET("/currency/rates", currencyHandler.StoreRatesFromCBR)          // api fetching
	r.GET("/currency/rate", currencyHandler.GetHistoricalRateByCharCode) // post req by char code n date

	// task sheduler
	c := cron.New()

	// every day 10 AM Moscow
	_, err = c.AddFunc("0 13 * * *", func() {
		log.Info("Auto updating valutes...")
		ctx := context.Background()
		err := currencyUsecase.FetchAndStoreRatesFromCBR(ctx)
		if err != nil {
			log.Errorf("Error by update valutes: %v", err)
		} else {
			log.Info("Successfylu update valutes")
		}
	})

	if err != nil {
		log.Fatalf("Error by add task to shedule: %v", err)
	}

	c.Start()
	log.Info("Sheduler initialized. Course updating every day in 10 AM")

	go func() {
		log.Info("Updating valutes...")
		time.Sleep(2 * time.Second)
		ctx := context.Background()
		err := currencyUsecase.FetchAndStoreRatesFromCBR(ctx)
		if err != nil {
			log.Errorf("Error updating valutes by server start: %v", err)
		} else {
			log.Info("Successfuly updatet valutes by server start")
		}
	}()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	go func() {
		log.Info("Server starting on port 8080...")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Got shutdown signal...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Error server shutdown:", err)
	}
	log.Info("Server stopped")

	c.Stop()
	log.Info("Sheduler stopped")

	log.Info("Gracefuly shutdowned")
}
