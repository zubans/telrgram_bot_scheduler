package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/yourusername/telegram-pin-forwarder/internal/database"
	"github.com/yourusername/telegram-pin-forwarder/internal/parser"
)

type Forwarder struct {
	bot        *tgbotapi.BotAPI
	repository *database.Repository
	daysAhead  int
	token      string
}

func NewForwarder(token string, repo *database.Repository, daysAhead int) (*Forwarder, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать бота: %w", err)
	}

	log.Printf("Авторизован как %s", bot.Self.UserName)
	return &Forwarder{
		bot:        bot,
		repository: repo,
		daysAhead:  daysAhead,
		token:      token,
	}, nil
}

func (f *Forwarder) GetPinnedMessage(chatID int64) (*tgbotapi.Message, error) {
	chatConfig := tgbotapi.ChatInfoConfig{
		ChatConfig: tgbotapi.ChatConfig{
			ChatID: chatID,
		},
	}

	chat, err := f.bot.GetChat(chatConfig)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить информацию о чате: %w", err)
	}

	log.Printf("Информация о чате получена: тип=%s, название=%s", chat.Type, chat.Title)

	if chat.PinnedMessage != nil {
		log.Println("Закрепленное сообщение найдено через GetChat")
		log.Printf("ID закрепленного сообщения: %d", chat.PinnedMessage.MessageID)
		return chat.PinnedMessage, nil
	}

	log.Println("Поле PinnedMessage пустое в GetChat, пытаемся получить через прямой HTTP запрос...")

	pinnedMsg, err := f.getPinnedMessageViaHTTP(chatID)
	if err == nil && pinnedMsg != nil {
		log.Println("Закрепленное сообщение найдено через прямой HTTP запрос")
		return pinnedMsg, nil
	}
	if err != nil {
		log.Printf("Ошибка при получении через HTTP: %v", err)
	}

	log.Println("Попытка получить информацию о чате через альтернативный метод...")

	chatMemberConfig := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: f.bot.Self.ID,
		},
	}

	member, err := f.bot.GetChatMember(chatMemberConfig)
	if err != nil {
		log.Printf("Не удалось получить информацию о членстве бота: %v", err)
	} else {
		log.Printf("Статус бота в чате: %s", member.Status)
		if member.Status == "left" || member.Status == "kicked" {
			return nil, fmt.Errorf("бот не является участником чата %d (статус: %s)", chatID, member.Status)
		}
	}

	return nil, fmt.Errorf("в чате %d не найдено закрепленного сообщения. Убедитесь, что: 1) сообщение закреплено в группе, 2) бот является участником и имеет права на чтение", chatID)
}

func (f *Forwarder) ForwardPinnedMessage(ctx context.Context, groupChatID int64) error {
	log.Printf("Получаем закрепленное сообщение из группы %d...", groupChatID)

	pinnedMessage, err := f.GetPinnedMessage(groupChatID)
	if err != nil {
		return err
	}

	if pinnedMessage.Text == "" {
		return fmt.Errorf("закрепленное сообщение не содержит текста")
	}

	log.Println("Закрепленное сообщение найдено!")
	log.Printf("Парсим список событий (проверяем события в течение %d дней)...", f.daysAhead)

	events := parser.ParseEventList(pinnedMessage.Text)
	log.Printf("Распарсено событий: %d", len(events))

	upcomingEvents := parser.GetUpcomingEvents(events, f.daysAhead)

	if len(upcomingEvents) == 0 {
		log.Println("Нет предстоящих событий в течение указанного периода")
		return nil
	}

	log.Printf("Найдено предстоящих событий: %d", len(upcomingEvents))

	var newEvents []*parser.EventEntry
	for _, event := range upcomingEvents {
		eventHash := database.GenerateEventHash(event.Date, event.Description)
		isSent, err := f.repository.IsEventSent(ctx, eventHash)
		if err != nil {
			log.Printf("Ошибка при проверке отправленного события: %v", err)
			continue
		}
		if isSent {
			log.Printf("Событие уже было отправлено: %s - %s", event.Date.Format("2006-01-02"), event.Description)
			continue
		}
		newEvents = append(newEvents, event)
	}

	if len(newEvents) == 0 {
		log.Println("Все предстоящие события уже были отправлены ранее")
		return nil
	}

	log.Printf("Новых событий для отправки: %d", len(newEvents))

	eventMessages := make([]string, 0)
	for _, event := range newEvents {
		formatted := parser.FormatEventForMessage(event)
		if formatted != "" {
			eventMessages = append(eventMessages, formatted)
		}
		log.Printf("  - %s", event.RawDate)
	}

	messageText := "🎉 Напоминание о предстоящих событиях:\n\n" + strings.Join(eventMessages, "\n")

	recipients, err := f.repository.GetActiveRecipients(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения списка получателей: %w", err)
	}

	if len(recipients) == 0 {
		log.Println("Нет активных получателей с разрешением на отправку")
		return nil
	}

	log.Printf("Найдено получателей с разрешением на отправку: %d", len(recipients))
	for _, recipient := range recipients {
		log.Printf("  - Пользователь ID: %d, Username: %s", recipient.UserID, recipient.Username)
	}

	log.Printf("Отправляем напоминание %d получателям из БД...", len(recipients))

	successCount := 0
	for _, recipient := range recipients {
		if err := f.sendMessage(recipient.UserID, messageText); err != nil {
			log.Printf("Ошибка при отправке пользователю %d: %v", recipient.UserID, err)
			errMsg := err.Error()
			f.repository.UpdateDeliveryStatus(ctx, recipient.UserID, "failed", &errMsg)
			continue
		}

		log.Printf("Напоминание отправлено пользователю %d", recipient.UserID)
		f.repository.UpdateDeliveryStatus(ctx, recipient.UserID, "success", nil)
		successCount++
		time.Sleep(500 * time.Millisecond)
	}

	if successCount > 0 {
		for _, event := range newEvents {
			eventHash := database.GenerateEventHash(event.Date, event.Description)
			if err := f.repository.MarkEventAsSent(ctx, event.Date, event.Description, eventHash); err != nil {
				log.Printf("Ошибка при сохранении информации об отправленном событии: %v", err)
			} else {
				log.Printf("Событие помечено как отправленное: %s - %s", event.Date.Format("2006-01-02"), event.Description)
			}
		}
	}

	f.repository.CreateMessageLog(ctx, pinnedMessage.MessageID, "event_reminder", messageText, len(recipients), successCount)

	log.Printf("Готово! Напоминания отправлены: %d/%d", successCount, len(recipients))
	return nil
}

func (f *Forwarder) getPinnedMessageViaHTTP(chatID int64) (*tgbotapi.Message, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getChat", f.token)

	payload := map[string]interface{}{
		"chat_id": chatID,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ошибка маршалинга JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	var apiResponse struct {
		OK          bool                   `json:"ok"`
		Result      map[string]interface{} `json:"result"`
		Description string                 `json:"description,omitempty"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		log.Printf("Полный ответ API: %s", string(body))
		return nil, fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if !apiResponse.OK {
		return nil, fmt.Errorf("API вернул ошибку: %s", apiResponse.Description)
	}

	if pinnedMsgData, exists := apiResponse.Result["pinned_message"]; exists && pinnedMsgData != nil {
		pinnedMsgBytes, err := json.Marshal(pinnedMsgData)
		if err != nil {
			return nil, fmt.Errorf("ошибка маршалинга закрепленного сообщения: %w", err)
		}

		var pinnedMsg tgbotapi.Message
		if err := json.Unmarshal(pinnedMsgBytes, &pinnedMsg); err != nil {
			log.Printf("Не удалось распарсить закрепленное сообщение: %v, данные: %s", err, string(pinnedMsgBytes))
			return nil, fmt.Errorf("ошибка парсинга закрепленного сообщения: %w", err)
		}

		return &pinnedMsg, nil
	}

	log.Printf("Поле pinned_message отсутствует в ответе API. Доступные поля: %v", getKeys(apiResponse.Result))
	return nil, fmt.Errorf("закрепленное сообщение не найдено в ответе API")
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (f *Forwarder) sendMessage(userID int64, text string) error {
	msg := tgbotapi.NewMessage(userID, text)
	_, err := f.bot.Send(msg)
	return err
}
