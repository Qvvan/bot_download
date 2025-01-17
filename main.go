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
	originalFile := "original_video.mp4"
	outputFile := "video.mp4"

	if err := runCommand("yt-dlp", "-o", originalFile, url); err != nil {
		log.Printf("Ошибка загрузки видео: %v", err)
		return "", err
	}

	if err := runCommand("ffmpeg", "-i", originalFile, "-c:v", "libx264", "-preset", "fast", "-c:a", "aac", "-b:a", "128k", "-movflags", "+faststart", outputFile); err != nil {
		log.Printf("Ошибка перекодирования видео: %v", err)
		return "", err
	}

	os.Remove(originalFile)
	return outputFile, nil
}

type AudioExtractor struct{}

func (ae AudioExtractor) Extract(videoFile string) (string, error) {
	audioFile := "audio.mp3"
	if err := runCommand("ffmpeg", "-i", videoFile, "-q:a", "0", "-map", "a", audioFile); err != nil {
		log.Printf("Ошибка извлечения аудио: %v", err)
		return "", err
	}

	return audioFile, nil
}

var callbackStorage = struct {
	sync.RWMutex
	data map[string]string
}{data: make(map[string]string)}

func generateCallbackID(data string) string {
	id := fmt.Sprintf("%x", md5.Sum([]byte(data)))
	callbackStorage.Lock()
	callbackStorage.data[id] = data
	callbackStorage.Unlock()
	return id
}

func getCallbackData(id string) (string, bool) {
	callbackStorage.RLock()
	data, exists := callbackStorage.data[id]
	callbackStorage.RUnlock()
	return data, exists
}

func cleanupCallbackStoragePeriodically() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		callbackStorage.Lock()
		for id := range callbackStorage.data {
			delete(callbackStorage.data, id)
		}
		callbackStorage.Unlock()
		log.Println("Callback storage очищен")
	}
}

func runCommand(command string, args ...string) error {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	log.Printf("Вывод команды: %s", string(output))
	return err
}

type BotHandler struct {
	Bot             *tgbotapi.BotAPI
	VideoDownloader Downloader
	AudioExtractor  AudioExtractor
}

// ForwardMessageToAdmin отправляет сообщение от пользователя админу
func (bh *BotHandler) ForwardMessageToAdmin(msg *tgbotapi.Message) {
	adminID := int64(323993202) // Ваш Telegram ID
	userID := msg.From.ID
	username := msg.From.UserName
	text := msg.Text
	// Формируем сообщение для администратора
	message := fmt.Sprintf(
		"Сообщение от пользователя:\nID: %d\nЮзернейм: @%s\nТекст: %s",
		userID, username, text,
	)
	// Отправляем сообщение админу
	adminMsg := tgbotapi.NewMessage(adminID, message)
	_, err := bh.Bot.Send(adminMsg)
	if err != nil {
		log.Printf("Ошибка отправки сообщения админу: %v", err)
	}
}

func (bh *BotHandler) StartBot() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bh.Bot.GetUpdatesChan(u)
	log.Println("Бот готов принимать обновления")

	var wg sync.WaitGroup
	for update := range updates {
		wg.Add(1)
		go func(update tgbotapi.Update) {
			defer wg.Done()
			if update.CallbackQuery != nil {
				bh.HandleCallback(update.CallbackQuery)
			} else if update.Message != nil {
			    bh.ForwardMessageToAdmin(update.Message)
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
		}(update)
	}
	wg.Wait()
}

func (bh *BotHandler) HandleStart(msg *tgbotapi.Message) {
	response := "Привет! Отправьте ссылку на видео, чтобы начать."
	message := tgbotapi.NewMessage(msg.Chat.ID, response)
	bh.Bot.Send(message)
}

func (bh *BotHandler) HandleDefault(msg *tgbotapi.Message) {
	response := "Я не понимаю эту команду. Отправьте ссылку на видео."
	message := tgbotapi.NewMessage(msg.Chat.ID, response)
	bh.Bot.Send(message)
}

func (bh *BotHandler) AskDownloadOption(msg *tgbotapi.Message) {
	url := sanitizeURL(msg.Text)
	if url == "" {
		bh.Bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Некорректная ссылка."))
		return
	}

	callbackID := generateCallbackID(url)
	inlineKeyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Скачать видео", "download_video:"+callbackID),
			tgbotapi.NewInlineKeyboardButtonData("Скачать аудио", "download_audio:"+callbackID),
		),
	)
	message := tgbotapi.NewMessage(msg.Chat.ID, "Выберите действие:")
	message.ReplyMarkup = inlineKeyboard
	bh.Bot.Send(message)
}

func (bh *BotHandler) HandleCallback(callback *tgbotapi.CallbackQuery) {
	data := callback.Data
	chatID := callback.Message.Chat.ID
	messageID := callback.Message.MessageID

	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Некорректный запрос."))
		return
	}

	action, callbackID := parts[0], parts[1]
	url, exists := getCallbackData(callbackID)
	if !exists {
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Данные не найдены."))
		return
	}

	go func() {
		statusMsg := tgbotapi.NewMessage(chatID, "")
		bh.Bot.Request(tgbotapi.NewDeleteMessage(chatID, messageID))
		switch action {
		case "download_video":
			statusMsg.Text = "Загружаем видео..."
			msg, _ := bh.Bot.Send(statusMsg)
			bh.HandleDownloadVideo(chatID, url, msg)
		case "download_audio":
			statusMsg.Text = "Извлекаем аудио..."
			msg, _ := bh.Bot.Send(statusMsg)
			bh.HandleDownloadAudio(chatID, url, msg)
		default:
			bh.Bot.Send(tgbotapi.NewMessage(chatID, "Неизвестное действие."))
		}
	}()
}

func (bh *BotHandler) HandleDownloadVideo(chatID int64, url string, message tgbotapi.Message) {
	videoFile, err := bh.VideoDownloader.Download(url)
	if err != nil {
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Ошибка загрузки видео."))
		return
	}

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, message.MessageID)
	if _, err := bh.Bot.Request(deleteMsg); err != nil {
		log.Printf("Ошибка удаления сообщения: %v", err)
	}

	bh.Bot.Send(tgbotapi.NewDocument(chatID, tgbotapi.FilePath(videoFile)))
	os.Remove(videoFile)
}

func (bh *BotHandler) HandleDownloadAudio(chatID int64, url string, message tgbotapi.Message) {
	videoFile, err := bh.VideoDownloader.Download(url)
	if err != nil {
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Ошибка загрузки видео."))
		return
	}

	deleteMsg := tgbotapi.NewDeleteMessage(chatID, message.MessageID)
	if _, err := bh.Bot.Request(deleteMsg); err != nil {
		log.Printf("Ошибка удаления сообщения: %v", err)
	}

	audioFile, err := bh.AudioExtractor.Extract(videoFile)
	if err != nil {
		bh.Bot.Send(tgbotapi.NewMessage(chatID, "Ошибка извлечения аудио."))
		return
	}
	bh.Bot.Send(tgbotapi.NewDocument(chatID, tgbotapi.FilePath(audioFile)))
	os.Remove(videoFile)
	os.Remove(audioFile)
}

func sanitizeURL(input string) string {
	u, err := url.Parse(strings.TrimSpace(input))
	if err != nil {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func main() {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не задан")
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
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
