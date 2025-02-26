package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v2"
)

// Config структура для хранения настроек
type Config struct {
	MediaType  string `yaml:"media_type"`
	DateFilter string `yaml:"date_filter"`
}

var config Config

// Загружаем настройки из файла config.yaml, если он существует
func loadConfig() {
	file, err := os.ReadFile("config.yaml")
	if err != nil {
		// Если файла нет, устанавливаем значения по умолчанию
		config = Config{
			MediaType:  "Видео",
			DateFilter: "Все ролики",
		}
		saveConfig()
		return
	}
	yaml.Unmarshal(file, &config)
}

// Сохраняем текущие настройки в config.yaml
func saveConfig() {
	data, _ := yaml.Marshal(&config)
	os.WriteFile("config.yaml", data, 0644)
}

// Функция для скачивания видео и аудио
func downloadMedia(w fyne.Window, url, folderName, dateAfter, mediaType string, currentNameLabel *widget.Label, totalLabel *widget.Label) {
	go func() {
		// Создаем папки для видео и аудио
		videoFolder := folderName + "/video"
		audioFolder := folderName + "/audio"

		// Создаем директории, если они не существуют
		os.MkdirAll(videoFolder, os.ModePerm)
		os.MkdirAll(audioFolder, os.ModePerm)

		// Основные аргументы для получения списка ссылок
		args := []string{"--flat-playlist", "--print", "url", "--no-cache-dir", "--limit-rate", "7M"}
		if dateAfter != "" && dateAfter != "Все ролики" {
			if len(dateAfter) != 8 || !isValidDate(dateAfter) {
				fmt.Println("Ошибка: неверный формат даты. Ожидается YYYYMMDD.")
				return
			}
			args = append(args, "--dateafter="+dateAfter)
		}

		// Получаем список ссылок на видео
		cmdList := exec.Command("./yt-dlp.exe", append(args, url)...)
		output, err := cmdList.CombinedOutput()
		if err != nil {
			fmt.Printf("Ошибка при получении списка видео %s: %v\n", url, err)
			return
		}

		// Преобразуем output (байтовый срез) в строку
		outputStr := string(output)

		// Разделяем строку на URL-адреса по символу новой строки
		videoURLs := strings.Split(outputStr, "\n")

		// Обновляем метку общего количества роликов
		totalLabel.SetText(fmt.Sprintf("Общее количество: %d", len(videoURLs)))

		// Скачиваем каждое видео или аудио
		for i, videoURL := range videoURLs {
			if videoURL == "" {
				continue
			}

			// Обновляем метку текущего скачиваемого ролика
			currentNameLabel.SetText(fmt.Sprintf("Скачивание %d из %d: %s", i+1, len(videoURLs), videoURL))

			// Аргументы для скачивания
			args := []string{"--limit-rate", "7M", "-f", "bestvideo+bestaudio/best"}
			if mediaType == "Аудио" {
				args = append(args, "-x", "--audio-format", "mp3")
			}

			// Формируем путь для сохранения
			outputTemplate := fmt.Sprintf("%s/%%(title)s.%%(ext)s", videoFolder)
			if mediaType == "Аудио" {
				outputTemplate = fmt.Sprintf("%s/%%(title)s.%%(ext)s", audioFolder)
			}
			args = append(args, "-o", outputTemplate, videoURL)

			// Запускаем скачивание
			cmd := exec.Command("./yt-dlp.exe", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err := cmd.Run()
			if err != nil {
				fmt.Printf("Ошибка при скачивании %s: %v\n", videoURL, err)
			}
		}

		// Обновляем метку после завершения
		currentNameLabel.SetText("Скачивание завершено!")
	}()
}

// Функция для проверки правильности формата даты
func isValidDate(date string) bool {
	for _, ch := range date {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// Функция для извлечения имени канала из URL
func getFolderNameFromURL(url string) string {
	// Используем регулярное выражение для извлечения имени канала
	re := regexp.MustCompile(`@([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(url)

	if len(matches) > 1 {
		// Возвращаем имя канала без специальных символов
		return matches[1]
	}
	return "default" // Если не удалось извлечь имя, возвращаем default
}

func main() {
	loadConfig()

	// Создаем приложение Fyne
	a := app.New()
	w := a.NewWindow("YouTube Downloader")
	w.Resize(fyne.NewSize(600, 400))

	// Поле ввода URL канала
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Введите URL канала")

	// Метка для отображения текущего скачиваемого ролика
	currentNameLabel := widget.NewLabel("Ожидание начала скачивания...")

	// Метка для отображения общего прогресса
	totalLabel := widget.NewLabel("0 / 0")

	// Функция для создания радиогруппы выбора типа медиа
	createMediaGroup := func(selected string) *widget.RadioGroup {
		mediaGroup := widget.NewRadioGroup([]string{"Видео", "Аудио"}, nil)
		mediaGroup.Horizontal = true
		mediaGroup.SetSelected(selected)
		mediaGroup.OnChanged = func(value string) {
			config.MediaType = value
			saveConfig()
		}
		return mediaGroup
	}

	// Создаем радиогруппу с начальным значением из конфигурации
    mediaGroup := createMediaGroup(config.MediaType)
    
    // Заголовок для выбора типа медиа
    mediaLabel := widget.NewLabel("Выберите тип медиа:")
    
    // Поле ввода даты (отображается только при выборе "Начальная дата")
    dateEntry := widget.NewEntry()
    dateEntry.SetPlaceHolder("YYYYMMDD")
    
    // Контейнер для поля ввода даты, растягиваем на весь экран
    dateContainer := container.NewVBox(dateEntry) // Используем VBox для растяжения
    dateEntry.Resize(fyne.NewSize(w.Content().Size().Width, 30)) // Растягиваем на всю ширину окна
    
    // Добавление контейнера в окно
    if config.DateFilter == "Начальная дата" {
        dateContainer.Show()
    } else {
        dateContainer.Hide()
    }
    
    // Функция для создания радиогруппы выбора периода
    createDateGroup := func(selected string, dateContainer *fyne.Container) *widget.RadioGroup {
        dateGroup := widget.NewRadioGroup([]string{"Все ролики", "Начальная дата"}, func(value string) {
            config.DateFilter = value
            if value == "Начальная дата" {
                dateContainer.Show()
            } else {
                dateContainer.Hide()
            }
            saveConfig()
        })
        dateGroup.SetSelected(selected)
        return dateGroup
    }

	// Создаем радиогруппу с начальным значением из конфигурации
	dateGroup := createDateGroup(config.DateFilter, dateContainer)

	// Кнопка для начала скачивания
	downloadButton := widget.NewButton("Начать скачивание", func() {
		url := urlEntry.Text
		if url == "" {
			fmt.Println("Ошибка: URL не указан")
			return
		}
		folderName := getFolderNameFromURL(url)
		downloadMedia(w, url, folderName, dateEntry.Text, config.MediaType, currentNameLabel, totalLabel)
	})

	// Добавляем все элементы на окно
	w.SetContent(container.NewVBox(
		urlEntry,
		mediaLabel,
		mediaGroup,
		dateGroup,
		dateContainer,
		downloadButton,
		currentNameLabel,
		totalLabel,
	))

	// Показываем окно и запускаем приложение
	w.ShowAndRun()
}
