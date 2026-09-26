package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/microsoft/go-mssqldb"
	"go.uber.org/zap"

	"github.com/example/go-data-profiler/internal/api"
	"github.com/example/go-data-profiler/internal/config"
	"github.com/example/go-data-profiler/internal/store"
	"github.com/example/go-data-profiler/internal/worker"
)

func main() {
	cfg:=config.Load()
	log,_:=zap.NewProduction()
	defer log.Sync()
	var db *sql.DB
	var err error
	if cfg.ProfilerDatabaseURL!=""{
		db,err=sql.Open("pgx",cfg.ProfilerDatabaseURL)
		if err!=nil{log.Fatal("open metadata db",zap.Error(err))}
		db.SetMaxOpenConns(10);db.SetMaxIdleConns(5);db.SetConnMaxLifetime(30*time.Minute)
		ctx,cancel:=context.WithTimeout(context.Background(),5*time.Second);err=db.PingContext(ctx);cancel()
		if err!=nil{log.Fatal("ping metadata db",zap.Error(err))}
	}
	if db==nil{log.Fatal("DATABASE_URL is required")}
	st:=store.NewPostgresStore(db)
	jobs:=worker.NewManager(st,cfg.JobWorkers,log,cfg.JobStaleAfter)
	jobs.Start()
	defer jobs.Stop()
	router:=api.NewRouter(api.Dependencies{Store:st,Jobs:jobs,Logger:log})
	srv:=&http.Server{Addr:cfg.HTTPAddr,Handler:router,ReadHeaderTimeout:5*time.Second,ReadTimeout:15*time.Second,WriteTimeout:60*time.Second,IdleTimeout:60*time.Second}
	go func(){log.Info("server started",zap.String("addr",cfg.HTTPAddr));if err:=srv.ListenAndServe();err!=nil&&!errors.Is(err,http.ErrServerClosed){log.Fatal("http server",zap.Error(err))}}()
	sig:=make(chan os.Signal,1);signal.Notify(sig,syscall.SIGTERM,syscall.SIGINT);<-sig
	ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second);defer cancel();_=srv.Shutdown(ctx);_=db.Close()
}
