package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"link-checker/internal/checker"
	"link-checker/internal/pdf"
	"link-checker/internal/storage"

	"github.com/gorilla/mux"
)

type Handler struct {
	router  *mux.Router
	store   *storage.Storage
	checker *checker.Checker
}

func NewHandler(store *storage.Storage) *Handler {
	h := &Handler{
		router:  mux.NewRouter(),
		store:   store,
		checker: checker.NewChecker(store),
	}

	h.setupRoutes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) setupRoutes() {
	h.router.HandleFunc("/api/check", h.handleCheckLinks).Methods("POST")
	h.router.HandleFunc("/api/report", h.handleGetReport).Methods("POST")
	h.router.HandleFunc("/api/status/{id:[0-9]+}", h.handleGetStatus).Methods("GET")
	h.router.HandleFunc("/health", h.handleHealthCheck).Methods("GET")
}

func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

type CheckRequest struct {
	Links []string `json:"links"`
}

type CheckResponse struct {
	Links    map[string]string `json:"links"`
	LinksNum int               `json:"links_num"`
}

func (h *Handler) handleCheckLinks(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if len(req.Links) == 0 {
		http.Error(w, "Список ссылок не может быть пустым", http.StatusBadRequest)
		return
	}

	// Добавляем задачу на проверку
	task := h.store.CreateTask(req.Links)

	// Запускаем проверку асинхронно
	go h.checker.CheckLinks(task)

	// Сразу возвращаем ответ с processing статусом
	respLinks := make(map[string]string)
	for _, link := range req.Links {
		respLinks[link] = "processing"
	}

	resp := CheckResponse{
		Links:    respLinks,
		LinksNum: task.ID,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Ошибка кодирования ответа", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Неверный ID", http.StatusBadRequest)
		return
	}

	task := h.store.GetTask(id)
	if task == nil {
		http.Error(w, "Задача не найдена", http.StatusNotFound)
		return
	}

	// Преобразуем для ответа
	result := make(map[string]string)
	for link, status := range task.Links {
		switch status {
		case storage.StatusAvailable:
			result[link] = "available"
		case storage.StatusUnavailable:
			result[link] = "not available"
		case storage.StatusProcessing:
			result[link] = "processing"
		default:
			result[link] = "error"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"links":     result,
		"links_num": task.ID,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Ошибка кодирования ответа", http.StatusInternalServerError)
		return
	}
}

type ReportRequest struct {
	LinksList []int `json:"links_list"`
}

func (h *Handler) handleGetReport(w http.ResponseWriter, r *http.Request) {
	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if len(req.LinksList) == 0 {
		http.Error(w, "Список ID не может быть пустым", http.StatusBadRequest)
		return
	}

	// Получаем данные для отчета
	reportData := h.store.GetTasksForReport(req.LinksList)
	if len(reportData) == 0 {
		http.Error(w, "Нет данных для указанных ID", http.StatusNotFound)
		return
	}

	// Генерируем PDF (используем правильное имя функции)
	pdfBytes, err := pdf.GenerateReport(reportData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка генерации отчета: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "attachment; filename=report.pdf")
	if _, err := w.Write(pdfBytes); err != nil {
		log.Printf("Ошибка отправки PDF: %v", err)
	}
}
