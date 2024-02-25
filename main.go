package main

import (
	"log"
	"path/filepath"

	"github.com/go-telegram-bot-api/telegram-bot-api"
)

var FolkTales = map[string]string{
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

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() && update.Message.Command() == "start" {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Привет! тут ты можешь послушать народные сказки!")
			msg.ReplyMarkup = folkTalesKeyboard()
			_, err := bot.Send(msg)
			if err != nil {
				log.Println("проблема с отправкой текста:", err)
			}
		} else {

			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Загружается...")
			sentMsg, err := bot.Send(msg)
			if err != nil {
				log.Println("Проблема с текстом загрузки:", err)
				continue
			}

			selectedFolkTale := update.Message.Text

			log.Println("Выбранная сказка:", selectedFolkTale)

			audioFilePath, ok := FolkTales[selectedFolkTale]
			if !ok {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Что-то пошло не так...")
				msg.ReplyMarkup = folkTalesKeyboard()
				_, err := bot.Send(msg)
				if err != nil {
					log.Println("пробелма с отправкой текста:", err)
				}
				continue
			}

			ext := filepath.Ext(audioFilePath)
			if ext != ".mp3" && ext != ".m4a" && ext != ".ogg" && ext != ".flac" {
				log.Println("Такой тип файла не поддерживается:", ext)
				continue
			}

			audioConfig := tgbotapi.NewAudioUpload(update.Message.Chat.ID, audioFilePath)
			_, err = bot.Send(audioConfig)
			if err != nil {
				log.Println("Проблема с отправкой аудио файла:", err)
				removeMsg := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, sentMsg.MessageID)
				_, err := bot.Send(removeMsg)
				if err != nil {
					log.Println("проблема удаления текста загрузки:", err)
				}
				continue
			}

			removeMsg := tgbotapi.NewDeleteMessage(update.Message.Chat.ID, sentMsg.MessageID)
			_, err = bot.Send(removeMsg)
			if err != nil {
				log.Println("проблема удаления текста загрузки:", err)
			}

		}
	}
}

func folkTalesKeyboard() tgbotapi.ReplyKeyboardMarkup {
	var buttons [][]tgbotapi.KeyboardButton

	for tale := range FolkTales {
		btn := tgbotapi.NewKeyboardButton(tale)
		buttons = append(buttons, []tgbotapi.KeyboardButton{btn})
	}

	return tgbotapi.NewReplyKeyboard(buttons...)
}
