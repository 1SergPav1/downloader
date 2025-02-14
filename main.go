package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Config struct {
	URLs []string `json:"urls"`
}

const (
	configPath = "./config.json"
)

var (
	size int64
	mu   sync.Mutex
)

func main() {
	config, err := readConfig(configPath)
	if err != nil {
		log.Fatal("Не удалось открыть config.json")
	}

	path := createFolder()
	if path == "" {
		log.Fatal("Не создать директорию для скачивания")
	}

	var wg sync.WaitGroup

	for _, url := range config.URLs {
		wg.Add(1) // Увеличиваем счетчик запущенных горутин на 1
		go downloadFile(url, path, &wg)
	}
	wg.Wait() // Ждем когда счетчик открытых горутин обнулится, прежде чем пойти дальше

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
func downloadFile(url, folder string, wg *sync.WaitGroup) {
	defer wg.Done() // По завершении функции счетчик запущенных горутин в wg будет уменьшен на 1 (wg - указатель на WaitGroup для синхронизации горутин).

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

	mu.Lock() // блокируем доступ к size, чтобы избежать гонки
	size += wr
	mu.Unlock()
}
