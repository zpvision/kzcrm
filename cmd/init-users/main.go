package main

import (
	"context"
	"log"
	"os"

	"kzcrm/internal/app"
)

func main() {
	if err := app.InitUsers(context.Background()); err != nil {
		log.Fatal(err)
	}
	log.Println("Пользователи созданы или обновлены")
	_ = os.Stdout.Sync()
}
