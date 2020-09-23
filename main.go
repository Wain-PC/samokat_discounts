package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
)

type Product struct {
	Name          string   `json:"name"`
	Images        []string `json:"imageUrl"`
	Price         int      `json:"price"`
	OldPrice      int      `json:"oldPrice"`
	Specification string   `json:"specification"`
	Discount      int
}

type Response struct {
	Products map[string]Product `json:"products"`
}

func getJson(url string, target interface{}) error {
	r, err := http.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}

func checkDiscounts(lat float64, lon float64, minDiscount int) ([]Product, error) {
	if lat == 0 {
		return nil, errors.New("can't get SAMOKAT_LAT from env variables")
	}
	if lon == 0 {
		return nil, errors.New("can't get SAMOKAT_LON from env variables")
	}

	url := fmt.Sprintf("https://api.samokat.ru/showcase/showcases?version=0&lat=%f&lon=%f", lat, lon)

	response := Response{}
	err := getJson(url, &response)

	if err != nil {
		return nil, err
	}

	slice := make([]Product, 0, len(response.Products))

	for _, value := range response.Products {
		value.Price = value.Price / 100
		value.OldPrice = value.OldPrice / 100
		if value.Price > 0 && value.OldPrice > 0 {
			value.Discount = 100 * (value.OldPrice - value.Price) / value.OldPrice
			if value.Discount >= minDiscount {
				slice = append(slice, value)
			}
		}

	}

	// Sort by discount
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Discount > slice[j].Discount
	})

	return slice, nil
}

func sendTelegram(botToken string, chatId string, text string) error {
	if botToken == "" {
		return errors.New("can't get TELEGRAM_APITOKEN from env variables")
	}
	if chatId == "" {
		return errors.New("can't get TELEGRAM_CHAT_ID from env variables")
	}

	chatIdInt64, err := strconv.ParseInt(chatId, 10, 64)
	if err != nil {
		return err
	}

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return err
	}

	log.Printf("Authorized on bot account %s", bot.Self.UserName)

	msg := tgbotapi.NewMessage(chatIdInt64, text)

	_, err = bot.Send(msg)

	return err
}

func createOutput(products []Product) string {
	str := new(strings.Builder)

	for i := 0; i < len(products); i++ {
		line := fmt.Sprintf("%s %d р., %d%%\n", products[i].Name, products[i].Price, products[i].Discount)
		_, _ = str.WriteString(line)
	}

	return str.String()
}

func main() {
	botToken := os.Getenv("TELEGRAM_APITOKEN")
	chatId := os.Getenv("TELEGRAM_CHAT_ID")
	minDiscount, err := strconv.Atoi(os.Getenv("SAMOKAT_MIN_DISCOUNT"))
	lat, err := strconv.ParseFloat(os.Getenv("SAMOKAT_LAT"), 64)
	lon, err := strconv.ParseFloat(os.Getenv("SAMOKAT_LON"), 64)

	products, err := checkDiscounts(lat, lon, minDiscount)

	if err != nil {
		panic(err)
	}

	outputString := createOutput(products)

	err = sendTelegram(botToken, chatId, outputString)

	if err != nil {
		panic(err)
	}

}
