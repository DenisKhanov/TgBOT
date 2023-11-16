package custom

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"log"
	"time"
)

type BotAPICustom struct {
	*tgbotapi.BotAPI // Встраивание оригинального API бота
}

func (cb *BotAPICustom) GetUpdatesChan(ctx context.Context, config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	ch := make(chan tgbotapi.Update, cb.Buffer)

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(ch)
				return
			default:
				updates, err := cb.GetUpdates(config)
				if err != nil {
					log.Println(err)
					log.Println("Failed to get updates, retrying in 3 seconds...")
					time.Sleep(time.Second * 3)

					continue
				}

				for _, update := range updates {
					if update.UpdateID >= config.Offset {
						config.Offset = update.UpdateID + 1
						ch <- update
					}
				}
			}
		}
	}()

	return ch
}
