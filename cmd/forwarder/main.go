package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"telegram_bot/telegram-pin-forwarder/internal/config"
	"telegram_bot/telegram-pin-forwarder/internal/database"
	"telegram_bot/telegram-pin-forwarder/internal/telegram"

	"github.com/robfig/cron/v3"
)

func main() {
	// Флаги командной строки
	initFlag := flag.Bool("init", false, "Инициализация конфигурации")
	onceFlag := flag.Bool("once", false, "Однократный запуск")
	flag.Parse()

	ctx := context.Background()

	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	// Если флаг -init, только инициализируем и выходим
	if *initFlag {
		log.Println("Конфигурация инициализирована и сохранена в config.yaml")
		return
	}

	// Проверяем обязательные параметры
	if cfg.Telegram.BotToken == "" {
		log.Fatal("Не указан токен бота (telegram.bot_token)")
	}
	if cfg.Telegram.GroupChatID == 0 {
		log.Fatal("Не указан ID группы (telegram.group_chat_id)")
	}

	// Проверяем количество дней для проверки событий
	if cfg.App.DaysAhead < 1 {
		cfg.App.DaysAhead = 5
		log.Printf("Используется значение по умолчанию для days_ahead: %d", cfg.App.DaysAhead)
	}

	// Подключаемся к базе данных
	db, err := database.NewDatabase(ctx, cfg.GetDatabaseURL())
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer db.Close()

	// Создаем репозиторий
	repo := database.NewRepository(db)

	// Добавляем пользователей из конфигурации в базу данных
	for _, userID := range cfg.Telegram.UserIDs {
		if err := repo.UpsertRecipient(ctx, userID, ""); err != nil {
			log.Printf("Предупреждение: не удалось добавить получателя %d: %v", userID, err)
		}
	}

	// Создаем форвардер
	forwarder, err := telegram.NewForwarder(cfg.Telegram.BotToken, repo, cfg.App.DaysAhead)
	if err != nil {
		log.Fatalf("Ошибка создания форвардера: %v", err)
	}

	log.Printf("Параметры приложения: проверяем события на %d дней вперед", cfg.App.DaysAhead)
	log.Printf("Режим работы: run_once=%v, schedule_cron=%s", cfg.App.RunOnce, cfg.App.ScheduleCron)

	// Если флаг -once или конфиг требует однократного запуска
	if *onceFlag || cfg.App.RunOnce {
		if err := forwarder.ForwardPinnedMessage(ctx, cfg.Telegram.GroupChatID); err != nil {
			log.Fatalf("Ошибка при отправке напоминаний: %v", err)
		}
		return
	}

	// Запускаем планировщик
	log.Println("Запуск планировщика...")
	log.Printf("Расписание: %s", cfg.App.ScheduleCron)

	c := cron.New()
	_, err = c.AddFunc(cfg.App.ScheduleCron, func() {
		log.Println("Выполнение запланированной задачи...")
		if err := forwarder.ForwardPinnedMessage(ctx, cfg.Telegram.GroupChatID); err != nil {
			log.Printf("Ошибка при отправке напоминаний: %v", err)
		}
	})

	if err != nil {
		log.Fatalf("Ошибка при добавлении задачи в планировщик: %v", err)
	}

	c.Start()
	log.Println("Приложение запущено. Ожидание запланированных задач...")

	// Обработка сигналов для корректного завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Завершение работы...")
	c.Stop()
}
