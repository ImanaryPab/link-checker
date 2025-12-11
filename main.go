package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"link-checker/internal/handler"
	"link-checker/internal/storage"
)

func main() {
	// Создаем директорию для сохранения состояния
	if err := os.MkdirAll("state", 0755); err != nil {
		log.Fatalf("Не удалось создать директорию state: %v", err)
	}

	// Инициализация хранилища
	store := storage.NewStorage()

	// Восстанавливаем состояние
	if err := store.RestoreState(); err != nil {
		log.Printf("Не удалось восстановить состояние: %v", err)
	}

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    ":8080",
		Handler: handler.NewHandler(store),
	}

	// Канал для graceful shutdown
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Получен сигнал остановки...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Сохраняем состояние перед остановкой
		if err := store.SaveState(); err != nil {
			log.Printf("Ошибка сохранения состояния: %v", err)
		}

		// Останавливаем сервер
		if err := server.Shutdown(ctx); err != nil {
			log.Fatalf("Ошибка при остановке сервера: %v", err)
		}

		close(done)
	}()

	log.Println("Сервер запущен на :8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}

	<-done
	log.Println("Сервер остановлен")
}
