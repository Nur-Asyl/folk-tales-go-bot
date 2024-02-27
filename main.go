package main

import (
	"folk-tales-module/database"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"path/filepath"
	"strings"
	"sync"
)

var (
	FolkTales = map[string]string{
		"царевна":                   "./audio/царевна.mp3",
		"каша из топора":            "./audio/кашаизтопора.mp3",
		"колобок":                   "./audio/колобок.mp3",
		"гуси лебеди":               "./audio/гуси лебеди.mp3",
		"иван царевич и серый волк": "./audio/иван царевич и серый волк.mp3",
		"илья муравец и соловей разбойник": "./audio/илья муравец и соловей разбойник.mp3",
		"маша и медведь":                   "./audio/маша и медведь.mp3",
		"репка":                            "./audio/репка.mp3",
		"сивка бурка":                      "./audio/сивка бурка.mp3",
		"три медведя":                      "./audio/три медведя.mp3",
	}
	waitingForFeedback  = make(map[int64]bool)
	waitingFeedbackLock sync.Mutex
	chatStates          = make(map[int64]*ChatState)
	reviews             = make(map[string][]Review)
)

type ChatState struct {
	SelectedFolkTale   string
	WaitingForFeedback bool
}

type Review struct {
	FolkTale   string
	ReviewText string
}

func populateReviewsFromDB(db *database.Database) {
	for folkTale := range FolkTales {
		feedbacks, err := db.GetFeedbacksByFolkTale(folkTale)
		if err != nil {
			log.Println("Ошибка при получении отзывов из базы данных:", err)
			continue
		}
		for _, feedback := range feedbacks {
			addReview(folkTale, feedback.Message)
		}
	}
}

func addReview(folkTale, reviewText string) {
	review := Review{
		FolkTale:   folkTale,
		ReviewText: reviewText,
	}
	// Append the new review to the slice of reviews for the given folk tale
	reviews[folkTale] = append(reviews[folkTale], review)
}

func main() {

	bot, err := tgbotapi.NewBotAPI("7023318213:AAHjg420XcRZ8wvVQ1zD_CgAUB-DBkoLRAM")
	if err != nil {
		log.Panic(err)
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, err := bot.GetUpdatesChan(updateConfig)
	if err != nil {
		log.Panic(err)
	}

	connectionString := "mongodb+srv://Nur-Asyl:min@cluster0.wxztznh.mongodb.net/?retryWrites=true&w=majority&appName=Cluster0"
	db, err := database.NewDatabase(connectionString, "folk-tales", "feedback")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()
	log.Println("Database connected")

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, db, update.Message)
		} else if update.CallbackQuery != nil {
			handleCallbackQuery(bot, update.CallbackQuery)
		}
	}
}

func handleMessage(bot *tgbotapi.BotAPI, db *database.Database, message *tgbotapi.Message) {
	if message.IsCommand() {
		handleCommand(bot, message)
	} else {
		if isWaitingForFeedback(message.Chat.ID) {
			handleFeedback(bot, db, message)
			resetFeedbackState(message.Chat.ID)
		} else {
			populateReviewsFromDB(db)
			handleFolkTale(bot, db, message)
		}
	}
}

func handleCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		sendStartMessage(bot, message.Chat.ID)
	case "feedback":
		sendFeedbackMessage(bot, message.Chat.ID)
		setFeedbackState(message.Chat.ID)
	case "reviews":
		handleReviewsCommand(bot, message)
	}
}

func sendStartMessage(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Привет! Тут ты можешь послушать народные сказки и поделиться своим мнением. Пожалуйста, выбери сказку:")
	msg.ReplyMarkup = folkTalesKeyboard()
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("Проблема с отправкой старта:", err)
	}
}

func sendFeedbackMessage(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Оставьте ваш отзыв:")
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("Проблема с отправкой запроса на отзыв:", err)
	}
}

func handleFolkTale(bot *tgbotapi.BotAPI, db *database.Database, message *tgbotapi.Message) {
	selectedFolkTale := strings.ToLower(message.Text)

	// Save selected folk tale in chat state
	chatID := message.Chat.ID
	chatState, ok := chatStates[chatID]
	if !ok {
		chatState = &ChatState{}
		chatStates[chatID] = chatState
	}
	chatState.SelectedFolkTale = selectedFolkTale

	msg := tgbotapi.NewMessage(message.Chat.ID, "Загружается...")
	sentMsg, err := bot.Send(msg)
	if err != nil {
		log.Println("Проблема с текстом загрузки:", err)
		return
	}

	audioFilePath, ok := FolkTales[selectedFolkTale]
	if !ok {
		msg := tgbotapi.NewMessage(message.Chat.ID, "К сожалению, такой сказки нет.")
		msg.ReplyMarkup = folkTalesKeyboard()
		_, err := bot.Send(msg)
		if err != nil {
			log.Println("Проблема с отправкой текста:", err)
		}
		return
	}

	ext := filepath.Ext(audioFilePath)
	if ext != ".mp3" && ext != ".m4a" && ext != ".ogg" && ext != ".flac" {
		log.Println("Такой тип файла не поддерживается:", ext)
		return
	}

	audioConfig := tgbotapi.NewAudioUpload(message.Chat.ID, audioFilePath)
	_, err = bot.Send(audioConfig)
	if err != nil {
		log.Println("Проблема с отправкой аудио файла:", err)
		removeMsg := tgbotapi.NewDeleteMessage(message.Chat.ID, sentMsg.MessageID)
		_, err := bot.Send(removeMsg)
		if err != nil {
			log.Println("Проблема удаления текста загрузки:", err)
			return
		}
	}

	btn := tgbotapi.NewInlineKeyboardButtonData("Оставить отзыв", "/feedback")
	msg = tgbotapi.NewMessage(message.Chat.ID, "Если хотите оставить отзыв, нажмите кнопку ниже.")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
	_, err = bot.Send(msg)
	if err != nil {
		log.Println("Проблема с отправкой текста:", err)
	}
}

func handleCallbackQuery(bot *tgbotapi.BotAPI, callbackQuery *tgbotapi.CallbackQuery) {
	switch callbackQuery.Data {
	case "/feedback":
		sendFeedbackMessage(bot, callbackQuery.Message.Chat.ID)
		setFeedbackState(callbackQuery.Message.Chat.ID)
	}
}

func handleFeedback(bot *tgbotapi.BotAPI, db *database.Database, message *tgbotapi.Message) {
	userID := message.From.ID
	chatID := message.Chat.ID

	// Retrieve selected folk tale from chat state
	chatState, ok := chatStates[chatID]
	if !ok {
		// Handle error
		return
	}
	selectedFolkTale := chatState.SelectedFolkTale

	feedback := strings.TrimPrefix(message.Text, "Отзыв:")
	err := db.SaveFeedback(int64(userID), selectedFolkTale, feedback)

	if err != nil {
		log.Println("Проблема с сохранением отзыва в базе данных:", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Извините, мы не смогли учесть ваш отзыв на данный момент. Пожалуйста, попробуйте еще раз позже.")
		_, err := bot.Send(msg)
		if err != nil {
			log.Println("Проблема с отправкой сообщения:", err)
		}
	} else {
		log.Println("Отзыв успешно сохранен в базе данных.")
		msg := tgbotapi.NewMessage(message.Chat.ID, "Спасибо за ваш отзыв!")
		_, err := bot.Send(msg)
		if err != nil {
			log.Println("Проблема с отправкой сообщения:", err)
		}
	}
}

func handleReviewsCommand(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	chatID := message.Chat.ID

	// Create message with reviews
	var reviewsText string
	for folkTale, reviews := range reviews {
		reviewsText += "Отзывы на сказку \"" + folkTale + "\":\n"
		for _, review := range reviews {
			reviewsText += "- " + review.ReviewText + "\n"
		}
		reviewsText += "\n"
	}

	// Send reviews to user
	msg := tgbotapi.NewMessage(chatID, reviewsText)
	_, err := bot.Send(msg)
	if err != nil {
		log.Println("Ошибка при отправке отзывов:", err)
	}
}

func setFeedbackState(userID int64) {
	waitingFeedbackLock.Lock()
	defer waitingFeedbackLock.Unlock()
	waitingForFeedback[userID] = true
}

func resetFeedbackState(userID int64) {
	waitingFeedbackLock.Lock()
	defer waitingFeedbackLock.Unlock()
	waitingForFeedback[userID] = false
}

func isWaitingForFeedback(userID int64) bool {
	waitingFeedbackLock.Lock()
	defer waitingFeedbackLock.Unlock()
	return waitingForFeedback[userID]
}

func folkTalesKeyboard() tgbotapi.ReplyKeyboardMarkup {
	var buttons [][]tgbotapi.KeyboardButton

	for tale := range FolkTales {
		btn := tgbotapi.NewKeyboardButton(tale)
		buttons = append(buttons, []tgbotapi.KeyboardButton{btn})
	}

	return tgbotapi.NewReplyKeyboard(buttons...)
}
