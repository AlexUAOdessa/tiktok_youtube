package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"gopkg.in/yaml.v2"
)

// Config структура для хранения настроек (тип медиа и фильтр даты)
type Config struct {
	MediaType  string `yaml:"media_type"`  // Тип медиа (Видео/Аудио)
	DateFilter string `yaml:"date_filter"` // Фильтр даты (Все ролики или Начальная дата)
	DateValue  string `yaml:"date_value"`  // Дата в формате YYYYMMDD
}

var config Config // Переменная для хранения конфигурации

// loadConfig загружает настройки из файла config.yaml
func loadConfig() {
	// Чтение файла конфигурации
	file, err := os.ReadFile("config.yaml")
	if err != nil {
		// Если файл не найден или ошибка, то создаем настройки по умолчанию
		config = Config{MediaType: "Видео", DateFilter: "Все ролики"}
		saveConfig()
		return
	}
	// Если файл есть, десериализуем его в структуру config
	yaml.Unmarshal(file, &config)
}

// saveConfig сохраняет текущие настройки в файл config.yaml
func saveConfig() {
	// Сериализуем структуру в YAML
	data, _ := yaml.Marshal(&config)
	// Записываем данные в файл
	os.WriteFile("config.yaml", data, 0644)
}

// terminateProcesses завершает все процессы yt-dlp.exe
func terminateProcesses() {
	cmd := exec.Command("taskkill", "/F", "/IM", "yt-dlp.exe") // Команда для завершения процессов yt-dlp.exe
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}    // Скрыть окно при выполнении команды
	cmd.Run()
}

// downloadMedia скачивает медиа файлы с канала по указанному URL
func downloadMedia(w fyne.Window, url, folderName, dateAfter, mediaType string, currentNameLabel, totalLabel *widget.Label) {
	go func() {
		// Папки для хранения видео и аудио
		videoFolder := folderName + "/video"
		audioFolder := folderName + "/audio"

		// Создаем папки для хранения файлов
		os.MkdirAll(videoFolder, os.ModePerm)
		os.MkdirAll(audioFolder, os.ModePerm)

		// Настройки для команды yt-dlp
		args := []string{"--flat-playlist", "--print", "url", "--no-cache-dir"} // Получаем список URL-ов видео
		if dateAfter != "" && dateAfter != "Все ролики" {
			// Если выбран фильтр по дате, проверяем корректность даты
			if len(dateAfter) != 8 || !isValidDate(dateAfter) {
				fmt.Println("Ошибка: неверный формат даты. Ожидается YYYYMMDD.")
				return
			}
			// Добавляем фильтр по дате
			args = append(args, "--dateafter="+dateAfter)
		}

		// Выполняем команду для получения списка URL-ов видео
		cmdList := exec.Command("./yt-dlp.exe", append(args, url)...)
		output, err := cmdList.CombinedOutput()
		if err != nil {
			// Если ошибка при получении списка видео, выводим ее
			fmt.Printf("Ошибка при получении списка видео %s: %v\n", url, err)
			return
		}

		// Разделяем результат выполнения команды на строки (URL-ы видео)
		videoURLs := strings.Split(string(output), "\n")
		totalLabel.SetText(fmt.Sprintf("Общее количество: %d", len(videoURLs)))

		// Определяем папку для скачивания (в зависимости от типа медиа)
		folder := videoFolder
		if mediaType == "Аудио" {
			folder = audioFolder
		}

		// Открываем файл names.txt для записи в конец
		namesFile, err := os.OpenFile(folder+"/names.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// Если ошибка при открытии файла, выводим ошибку
			fmt.Printf("Ошибка при открытии файла names.txt: %v\n", err)
			return
		}
		defer namesFile.Close()

		// Перебираем все URL-ы видео и скачиваем их
		for i, videoURL := range videoURLs {
			if videoURL == "" {
				continue // Пропускаем пустые строки
			}

			// Получаем название видео с помощью yt-dlp
			cmdTitle := exec.Command("./yt-dlp.exe", "--get-title", videoURL)
			titleOutput, err := cmdTitle.CombinedOutput()
			if err != nil {
				// Если ошибка при получении названия видео, выводим ошибку
				fmt.Printf("Ошибка при получении названия видео %s: %v\n", videoURL, err)
				continue
			}

			// Убираем возможный лишний пробел или символ новой строки в конце
			videoTitle := strings.TrimSpace(string(titleOutput))

			// Обновляем метку текущего скачиваемого видео
			currentNameLabel.SetText(fmt.Sprintf("Скачивание %d из %d: %s", i+1, len(videoURLs), videoTitle))

			// Подготовка аргументов для скачивания
			args := []string{"-f", "bestvideo+bestaudio/best", "--merge-output-format", "mp4"}
			if mediaType == "Аудио" {
				// Если выбран тип "Аудио", добавляем аргументы для скачивания только аудио
				args = []string{"-x", "--audio-format", "mp3", "--audio-quality", "0"}
			}

			// Добавляем параметры для скачивания в папку
			args = append(args, "-o", fmt.Sprintf("%s/%%(title)s.%%(ext)s", folder), videoURL)

			// Запуск команды для скачивания
			cmd := exec.Command("./yt-dlp.exe", args...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				// Если ошибка при скачивании, выводим ошибку
				fmt.Printf("Ошибка при скачивании %s: %v\n", videoTitle, err)
			}

			// Добавляем название видео в файл names.txt
			_, err = fmt.Fprintln(namesFile, videoTitle)
			if err != nil {
				// Если ошибка при записи в файл, выводим ошибку
				fmt.Printf("Ошибка при записи в файл names.txt: %v\n", err)
			}
		}

		// Обновляем метку о завершении скачивания
		currentNameLabel.SetText("Скачивание завершено!")
	}()
}

// isValidDate проверяет корректность даты в формате YYYYMMDD
func isValidDate(date string) bool {
	// Проверяем, что дата состоит только из цифр
	for _, ch := range date {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// getFolderNameFromURL извлекает имя папки из URL (например, имя канала)
func getFolderNameFromURL(url string) string {
	re := regexp.MustCompile(`@([a-zA-Z0-9_-]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		// Возвращаем имя канала или плейлиста из URL
		return matches[1]
	}
	return "default" // Если имя не найдено, возвращаем "default"
}

func main() {
	loadConfig() // Загружаем конфигурацию

	a := app.New() // Создаем новое приложение
	w := a.NewWindow("YouTube Downloader") // Создаем новое окно приложения
	w.Resize(fyne.NewSize(600, 400)) // Задаем размер окна

	// Создаем элементы интерфейса
	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("Введите URL канала")
	currentNameLabel := widget.NewLabel("Ожидание начала скачивания...")
	totalLabel := widget.NewLabel("0 / 0")

	// Группа для выбора типа медиа (Видео или Аудио)
	mediaGroup := widget.NewRadioGroup([]string{"Видео", "Аудио"}, nil)
	mediaGroup.Horizontal = true
	mediaGroup.SetSelected(config.MediaType)
	mediaGroup.OnChanged = func(value string) {
		// Сохраняем выбранный тип медиа в конфигурацию
		config.MediaType = value
		saveConfig()
	}

	// Группа для выбора фильтра даты
	dateGroup := widget.NewRadioGroup([]string{"Все ролики", "Начальная дата"}, nil)
	dateGroup.Horizontal = true
	dateGroup.SetSelected(config.DateFilter)

	// Создаем поле ввода даты, но скрываем его, если выбран "Все ролики"
	dateEntry := widget.NewEntry()
	dateEntry.SetPlaceHolder("Введите дату (YYYYMMDD)")
	dateEntry.Hide()

	// Обработчик для изменения выбора фильтра даты
	dateGroup.OnChanged = func(value string) {
		if value == "Все ролики" {
			// Если выбран "Все ролики", скрываем поле ввода даты
			dateEntry.Hide()
			// Очищаем значение даты
			config.DateValue = ""
		} else {
			// Если выбрана "Начальная дата", показываем поле ввода даты
			dateEntry.Show()
		}
		// Сохраняем выбранный фильтр даты в конфигурацию
		config.DateFilter = value
		saveConfig()
	}

	// Кнопка для начала скачивания
	downloadButton := widget.NewButton("Начать скачивание", func() {
		url := urlEntry.Text
		if url == "" {
			// Если URL не указан, выводим ошибку
			fmt.Println("Ошибка: URL не указан")
			return
		}

		// Получаем имя папки для скачивания
		folderName := getFolderNameFromURL(url)
		// Получаем значение даты, если оно указано
		dateAfter := config.DateValue
		if config.DateFilter == "Начальная дата" && dateEntry.Text != "" {
			dateAfter = dateEntry.Text
		}
		// Запускаем функцию скачивания
		downloadMedia(w, url, folderName, dateAfter, config.MediaType, currentNameLabel, totalLabel)
	})

	// Размещаем элементы интерфейса в окне
	w.SetContent(container.NewVBox(
		urlEntry,
		mediaGroup,
		dateGroup,
		dateEntry,
		downloadButton,
		currentNameLabel,
		totalLabel,
	))

	// При закрытии окна завершаем все процессы yt-dlp.exe
	w.SetOnClosed(func() {
		terminateProcesses()
	})

	// Запускаем приложение
	w.ShowAndRun()
}
