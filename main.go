package main

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Downloader interface {
	Download(url string) (string, error)
}

type VideoDownloader struct{}

func (vd VideoDownloader) Download(url string) (string, error) {
	log.Printf("Начинаем загрузку видео по URL: %s", url)

	originalFile := "original_video.mp4"
	outputFile := "video.mp4"

	// Скачиваем видео
	if err := runCommand("yt-dlp", "-o", originalFile, url); err != nil {
		log.Printf("Ошибка загрузки видео с yt-dlp: %v", err)
		return "", fmt.Errorf("Ошибка скачивания видео: %v", err)
	}
	log.Printf("Видео успешно загружено: %s", originalFile)

	// Перекодируем видео для совместимости
	if err := runCommand("ffmpeg", "-i", originalFile, "-c:v", "libx264", "-preset", "fast", "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", outputFile); err != nil {
		log.Printf("Ошибка перекодирования видео: %v", err)
		return "", fmt.Errorf("Ошибка перекодирования видео: %v", err)
	}
	log.Printf("Видео успешно перекодировано: %s", outputFile)

	// Удаляем исходный файл
	os.Remove(originalFile)

	return outputFile, nil
}

type AudioExtractor struct{}

func (ae AudioExtractor) Extract(videoFile string) (string, error) {
	log.Printf("Начинаем извлечение аудио из видео: %s", videoFile)

	audioFile := "audio.mp3"
	if err := runCommand("ffmpeg", "-i", videoFile, "-q:a", "0", "-map", "a", audioFile); err != nil {
		log.Printf("Ошибка извлечения аудио: %v", err)
		return "", fmt.Errorf("Ошибка извлечения аудио: %v", err)
	}
	log.Printf("Аудио успешно извлечено: %s", audioFile)

	return audioFile, nil
}

// Хранилище для callback data
var callbackStorage = struct {
	sync.RWMutex
	data map[string]string
}{data: make(map[string]string)}

// Генерация уникального идентификатора для callback data
func generateCallbackID(data string) string {
	id := fmt.Sprintf("%x", md5.Sum([]byte(data)))
	callbackStorage.Lock()
	callbackStorage.data[id] = data
	callbackStorage.Unlock()
	return id
}

// Получение callback data по идентификатору
func getCallbackData(id string) (string, bool) {
	callbackStorage.RLock()
	data, exists := callbackStorage.data[id]
	callbackStorage.RUnlock()
	return data, exists
}

// Периодическая очистка устаревших данных
func cleanupCallbackStoragePeriodically() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		callbackStorage.Lock()
		for id := range callbackStorage.data {
			delete(callbackStorage.data, id)
		}
		callbackStorage.Unlock()
	}
	log.Println("Callback storage очищен")
}

// Утилита для выполнения команд
func runCommand(command string, args ...string) error {
	log.Printf("Запускаем команду: %s %v", command, args)

	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	log.Printf("Вывод команды: %s", string(output))

	if err != nil {
		log.Printf("Ошибка выполнения команды: %v", err)
		return fmt.Errorf("команда завершилась с ошибкой: %v", err)
	}
	return nil
}

type BotHandler struct {
	Bot             *tgbotapi.BotAPI
	VideoDownloader Downloader
	AudioExtractor  AudioExtractor
}

func (bh *BotHandler) StartBot() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bh.Bot.GetUpdatesChan(u)

	log.Println("Бот готов принимать обновления")
	for update := range updates {
		if update.CallbackQuery != nil {
			log.Printf("Обработан CallbackQuery: %s", update.CallbackQuery.Data)
			bh.HandleCallback(update.CallbackQuery)
		} else if update.Message != nil {
			log.Printf("Обработано сообщение от пользователя: %s", update.Message.Text)
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					bh.HandleStart(update.Message)
				default:
					bh.HandleDefault(update.Message)
				}
			} else {
				bh.AskDownloadOption(update.Message)
			}
		}
	}
}

func (bh *BotHandler) HandleStart(msg *tgbotapi.Message) {
	response := "Привет! Я помогу скачать видео или аудио из Instagram. Отправьте ссылку и выберите действие."
	message := tgbotapi.NewMessage(msg.Chat.ID, response)
	bh.Bot.Send(message)
}

func (bh *BotHandler) HandleDefault(msg *tgbotapi.Message) {
	response := "Я не понимаю эту команду. Пожалуйста, отправьте ссылку на видео."
	message := tgbotapi.NewMessage(msg.Chat.ID, response)
	bh.Bot.Send(message)
}

func (bh *BotHandler) AskDownloadOption(msg *tgbotapi.Message) {
	url := sanitizeURL(msg.Text)
	if url == "" {
		log.Printf("Получено некорректное сообщение: %s", msg.Text)
		bh.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Пожалуйста, отправьте ссылку на видео."))
		return
	}

	log.Printf("Сформирована ссылка для скачивания: %s", url)
	callbackID := generateCallbackID(url)

	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Скачать видео", "download_video:"+callbackID),
			tgbotapi.NewInlineKeyboardButtonData("Скачать аудио", "download_audio:"+callbackID),
		),
	)

	message := tgbotapi.NewMessage(msg.Chat.ID, "Что вы хотите сделать?")
	message.ReplyMarkup = inlineKeyboard
	bh.Bot.Send(message)
}

func (bh *BotHandler) HandleCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		log.Printf("Получен некорректный CallbackQuery: %s", data)
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Некорректный запрос."))
		return
	}

	action, callbackID := parts[0], parts[1]
	url, exists := getCallbackData(callbackID)
	if !exists {
		log.Printf("Callback data не найдены: %s", callbackID)
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Ошибка: данные не найдены."))
		return
	}

	log.Printf("Начата обработка действия: %s для URL: %s", action, url)
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)
	if _, err := bh.Bot.Request(deleteMsg); err != nil {
		log.Printf("Ошибка удаления сообщения: %v", err)
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Не удалось удалить сообщение."))
		return
	}

	statusMsg := tgbotapi.NewMessage(chatID, "")
	switch action {
	case "download_video":
		statusMsg.Text = "Загружаем видео..."
		message, _ := bh.Bot.Send(statusMsg)
		bh.HandleDownloadVideo(chatID, url, message)
	case "download_audio":
		statusMsg.Text = "Загружаем аудио..."
		message, _ := bh.Bot.Send(statusMsg)
		bh.HandleDownloadAudio(chatID, url, message)
	default:
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Неизвестное действие."))
	}
}

func sanitizeURL(input string) string {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		log.Printf("Ошибка парсинга URL: %v", err)
		return input
	}
	u.RawQuery = "" // Убираем параметры запроса
	u.Fragment = "" // Убираем фрагмент, если есть
	cleaned := u.String()
	log.Printf("Очистка URL: %s -> %s", input, cleaned)
	return cleaned
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не задан")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Ошибка создания бота: %v", err)
	}

	go cleanupCallbackStoragePeriodically()

	handler := BotHandler{
		Bot:             bot,
		VideoDownloader: VideoDownloader{},
		AudioExtractor:  AudioExtractor{},
	}

	log.Printf("Бот запущен: @%s", bot.Self.UserName)
	handler.StartBot()
}
