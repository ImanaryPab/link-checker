package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

type LinkStatus string

const (
	StatusAvailable   LinkStatus = "available"
	StatusUnavailable LinkStatus = "unavailable"
	StatusProcessing  LinkStatus = "processing"
	StatusError       LinkStatus = "error"
)

type Task struct {
	ID        int                   `json:"id"`
	Links     map[string]LinkStatus `json:"links"`
	CreatedAt time.Time             `json:"created_at"`
	UpdatedAt time.Time             `json:"updated_at"`
}

type Storage struct {
	mu        sync.RWMutex
	tasks     map[int]*Task
	nextID    int
	stateFile string
}

func NewStorage() *Storage {
	return &Storage{
		tasks:     make(map[int]*Task),
		nextID:    1,
		stateFile: "state/storage.json",
	}
}

type savedState struct {
	Tasks  map[int]*Task `json:"tasks"`
	NextID int           `json:"next_id"`
}

func (s *Storage) SaveState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state := savedState{
		Tasks:  s.tasks,
		NextID: s.nextID,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("ошибка маршалинга состояния: %w", err)
	}

	if err := os.WriteFile(s.stateFile, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи в файл: %w", err)
	}

	log.Printf("Состояние сохранено. Задач: %d", len(s.tasks))
	return nil
}

func (s *Storage) RestoreState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.stateFile); os.IsNotExist(err) {
		log.Println("Файл состояния не найден, создаем новое хранилище")
		return nil
	}

	data, err := os.ReadFile(s.stateFile)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	var state savedState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("ошибка парсинга состояния: %w", err)
	}

	s.tasks = state.Tasks
	s.nextID = state.NextID
	log.Printf("Состояние восстановлено. Задач: %d", len(s.tasks))
	return nil
}

func (s *Storage) CreateTask(links []string) *Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &Task{
		ID:        s.nextID,
		Links:     make(map[string]LinkStatus),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	for _, link := range links {
		task.Links[link] = StatusProcessing
	}

	s.tasks[task.ID] = task
	s.nextID++

	log.Printf("Создана новая задача #%d с %d ссылками", task.ID, len(links))

	// Асинхронно сохраняем состояние
	go func() {
		if err := s.SaveState(); err != nil {
			log.Printf("Ошибка сохранения состояния: %v", err)
		}
	}()

	return task
}

func (s *Storage) UpdateLinkStatus(taskID int, link string, status LinkStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if task, exists := s.tasks[taskID]; exists {
		if _, linkExists := task.Links[link]; linkExists {
			task.Links[link] = status
			task.UpdatedAt = time.Now()
			log.Printf("Обновлен статус задачи #%d, ссылка: %s -> %s", taskID, link, status)
		}
	}
}

func (s *Storage) GetTask(taskID int) *Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.tasks[taskID]
}

func (s *Storage) GetTasksForReport(taskIDs []int) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var tasks []*Task
	for _, id := range taskIDs {
		if task, exists := s.tasks[id]; exists {
			tasks = append(tasks, task)
		}
	}
	log.Printf("Запрошены задачи для отчета: %v, найдено: %d", taskIDs, len(tasks))
	return tasks
}

func (s *Storage) GetAllTasks() []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}
