package main

import (
	"context"
	"log"
	"net"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"almoxarifados-a2g/internal/auth"
	"almoxarifados-a2g/internal/database"
	"almoxarifados-a2g/internal/routes"
)

func main() {

	_ = godotenv.Load()

	ctx := context.Background()
	db, err := database.Connect(ctx, dsn())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := criarAdminInicial(ctx, db); err != nil {
		log.Fatal(err)
	}

	app, err := routes.New(db, routes.Config{
		FrontendURL:  env("FRONTEND_URL", "http://localhost:5173"),
		CookieSecure: env("COOKIE_SECURE", "true") == "true",
	})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		_ = app.Shutdown()
	}()

	if err := app.Listen(":" + env("PORT", "3000")); err != nil {
		log.Fatal(err)
	}
}

func dsn() string {
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD")),
		Host:     net.JoinHostPort(env("DB_HOST", "localhost"), env("DB_PORT", "5432")),
		Path:     os.Getenv("DB_NAME"),
		RawQuery: "sslmode=" + env("DB_SSLMODE", "disable"),
	}
	return u.String()
}

func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

func criarAdminInicial(ctx context.Context, db *database.Store) error {
	n, err := db.CountUsuarios(ctx)
	if err != nil || n > 0 {
		return err
	}

	email, senha := os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")
	if email == "" || len(senha) < 8 {
		log.Println("nenhum usuário cadastrado: defina ADMIN_EMAIL e ADMIN_PASSWORD (mín. 8 caracteres) para criar o administrador")
		return nil
	}

	hash, err := auth.HashSenha(senha)
	if err != nil {
		return err
	}
	_, err = db.CreateUsuario(ctx, database.CreateUsuarioParams{
		Nome:      env("ADMIN_NAME", "Administrador"),
		Email:     email,
		Perfil:    "admin",
		SenhaHash: hash,
	})
	if err == nil {
		log.Printf("administrador inicial criado: %s", email)
	}
	return err
}
