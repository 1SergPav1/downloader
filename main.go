package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	URLs []string `json:"urls"`
}

const (
	configPath = "./config.json"
)

var size int64

func main() {
	config, err := readConfig(configPath)
	if err != nil {
		log.Fatal("Не удалось открыть config.json")
	}

	ch := make(chan int64, len(config.URLs)) // Создаем канал с буфером по количеству ссылок в config.json
	syncCh := make(chan bool)                // Канал куда мы запишем значение, когда все горутины закончат скачивание.

	path := createFolder()
	if path == "" {
		log.Fatal("Не создать директорию для скачивания")
	}

	for _, url := range config.URLs {
		go downloadFile(url, path, ch)
	}

	go func() { // Подсчет общего объема скачанных данных выполняется в отдельной горутине, по мере поступления данных в канал ch.
		for i := 0; i < len(config.URLs); i++ { // При достижении len(config.URLs) значений эта горутина закроет канал ch и запишет занчени в канал syncCh.
			size += <-ch
		}
		close(ch)
		syncCh <- true
	}()

	<-syncCh // Горутина main ждет значение в канале syncCh
	log.Printf("Общий объем: %d", size)
}

// Чтение конфига с ссылками
func readConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}
	return &config, nil
}

// Создание папки для скачивания
func createFolder() string {
	now := time.Now().Format(time.DateTime)
	path := filepath.Join("download", now)

	if err := os.Mkdir(path, os.ModePerm); err != nil {
		return ""
	}
	return path
}

func getFileName(url string) string {
	data := strings.Split(url, "/")
	return data[len(data)-1]
}

// Скачивание файла
func downloadFile(url, folder string, ch chan int64) {
	fileName := filepath.Join(folder, getFileName(url))

	resp, err := http.Get(url)
	if err != nil {
		log.Println("Ошибка скачивания файла " + url + " " + err.Error())
		return
	}
	defer resp.Body.Close()

	file, err := os.Create(fileName)
	if err != nil {
		log.Println("Ошибка создания файла " + fileName + " " + err.Error())
		return
	}
	defer file.Close()

	wr, err := io.Copy(file, resp.Body)
	if err != nil {
		log.Println("Ошибка сохранения в файл " + fileName + " " + err.Error())
		return
	}

	ch <- wr
	log.Printf("Горутина положила в канал %d", wr)
}
