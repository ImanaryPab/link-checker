package checker

import (
	"link-checker/internal/storage"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Checker struct {
	store      *storage.Storage
	httpClient *http.Client
	wg         sync.WaitGroup
}

func NewChecker(store *storage.Storage) *Checker {
	return &Checker{
		store: store,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     30 * time.Second,
			},
		},
	}
}

func (c *Checker) CheckLinks(task *storage.Task) {
	log.Printf("Начинаем проверку задачи #%d (%d ссылок)", task.ID, len(task.Links))

	var wg sync.WaitGroup

	for link := range task.Links {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			status := c.checkSingleLink(l)
			c.store.UpdateLinkStatus(task.ID, l, status)
		}(link)
	}

	wg.Wait()

	// Сохраняем состояние после проверки всех ссылок
	if err := c.store.SaveState(); err != nil {
		log.Printf("Ошибка сохранения состояния после проверки задачи #%d: %v", task.ID, err)
	}

	log.Printf("Завершена проверка задачи #%d", task.ID)
}

func (c *Checker) checkSingleLink(link string) storage.LinkStatus {
	// Добавляем схему, если отсутствует
	url := link
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Printf("Ошибка создания запроса для %s: %v", link, err)
		return storage.StatusError
	}

	// Устанавливаем User-Agent
	req.Header.Set("User-Agent", "Link-Checker/1.0")

	// Делаем запрос с таймаутом
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("Ссылка %s недоступна: %v (время: %v)", link, err, elapsed)
		return storage.StatusUnavailable
	}
	defer resp.Body.Close()

	log.Printf("Ссылка %s: статус %d (время: %v)", link, resp.StatusCode, elapsed)

	// Проверяем статус код
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return storage.StatusAvailable
	}

	return storage.StatusUnavailable
}

func (c *Checker) Stop() {
	c.wg.Wait()
}
