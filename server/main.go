package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Rate struct {
	ID          string `gorm:"primaryKey`
	Code        string
	Codein      string
	Name        string
	High        string
	Low         string
	VarBid      string
	PctChange   string
	Bid         string
	Ask         string
	Timestamp   string
	Create_date string
}

type Exchange struct {
	UsdBrl Rate
}

var getCurrentRateTimeout = 200 * time.Nanosecond
var saveRateInDatabaseTimeout = 100 * time.Millisecond
var lock = &sync.Mutex{}
var databaseInstance *gorm.DB

func main() {
	initDatabase()

	http.HandleFunc("/cotacao", handler)
	http.ListenAndServe(":8080", nil)
}

func getInstance() *gorm.DB {
	if databaseInstance == nil {
		lock.Lock()
		defer lock.Unlock()

		dsn := "root:root@tcp(localhost:3306)/challenge?charset=utf8mb4&parseTime=True&loc=Local"
		database, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			panic(err)
		}

		databaseInstance = database
	}

	return databaseInstance
}

func initDatabase() {
	database := getInstance()

	err := database.AutoMigrate(&Rate{})
	if err != nil {
		panic(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	exchange, err := getCurrentRate(ctx)
	if err != nil {
		http.Error(w, "Request Error", http.StatusInternalServerError)
	}

	err = insertRate(ctx, &exchange.UsdBrl)
	if err != nil {
		http.Error(w, "Request Error", http.StatusInternalServerError)
	}

	if errors.Is(err, context.DeadlineExceeded) {
		log.Println("Request Cancelled")
		return
	}

	w.Write([]byte(exchange.UsdBrl.Bid))

	select {
	case <-time.After(getCurrentRateTimeout):
		log.Println("Request Success")

	case <-ctx.Done():
		log.Println("Request Cancelled")
	}
}

func getCurrentRate(ctx context.Context) (Exchange, error) {
	ctx, cancel := context.WithTimeout(ctx, getCurrentRateTimeout)
	defer cancel()

	var exchange Exchange

	request, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return exchange, err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return exchange, err
	}
	defer response.Body.Close()

	responseJson, err := io.ReadAll(response.Body)
	if err != nil {
		return exchange, err
	}

	json.Unmarshal(responseJson, &exchange)

	select {
	case <-time.After(getCurrentRateTimeout):
		return exchange, nil

	case <-ctx.Done():
		return exchange, ctx.Err()
	}
}

func insertRate(ctx context.Context, rate *Rate) error {
	ctx, cancel := context.WithTimeout(ctx, saveRateInDatabaseTimeout)
	defer cancel()

	database := getInstance()

	database.WithContext(ctx).Create(&Rate{
		ID:          uuid.New().String(),
		Code:        rate.Code,
		Codein:      rate.Codein,
		Name:        rate.Name,
		High:        rate.High,
		Low:         rate.Low,
		VarBid:      rate.VarBid,
		PctChange:   rate.PctChange,
		Bid:         rate.Bid,
		Ask:         rate.Ask,
		Timestamp:   rate.Timestamp,
		Create_date: rate.Create_date,
	})

	if ctx.Done() != nil {
		return ctx.Err()
	}

	return nil

}
