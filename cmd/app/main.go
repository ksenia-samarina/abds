package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

// Domain Models

type User struct {
	ID        uint64 `json:"id"`
	FIO       string `json:"fio"`
	Phone     string `json:"phone"`
	PhonePrev string `json:"phone_prev"`
}

type Realty struct {
	ID      uint64 `json:"id"`
	Address string `json:"address"`
	Square  uint32 `json:"square"`
	Type    string `json:"type"`
	Number  string `json:"number"`
	Rooms   uint8  `json:"rooms"`
	Floor   uint8  `json:"floor"`
	Porch   uint8  `json:"porch"`
}

type MeterReading struct {
	ReadingDate string `json:"reading_date"`
	UserID      uint64 `json:"user_id"`
	TariffID    uint64 `json:"tariff_id"`
	GaugeID     uint64 `json:"gauge_id"`
	DayKW       uint32 `json:"day_kw"`
	NightKW     uint32 `json:"night_kw"`
}

type Invoice struct {
	TariffID  uint64 `json:"tariff_id"`
	GaugeID   uint64 `json:"gauge_id"`
	DayKW     uint32 `json:"day_kw"`
	NightKW   uint32 `json:"night_kw"`
	Price     uint64 `json:"price"`
	Deadline  string `json:"deadline"`
	CreatedAt string `json:"created_at"`
}

type Tariff struct {
	ID             uint64 `json:"id"`
	Name           string `json:"name"`
	PriceDayKWPH   uint32 `json:"price_day_kwph"`
	PriceNightKWPH uint32 `json:"price_night_kwph"`
	ValidFrom      string `json:"valid_from"`
	ValidTo        string `json:"valid_to"`
}

// Repositories

type DatabaseRepository interface {
	GetUser(ctx context.Context, id uint64) (*User, error)
	UpdateUser(ctx context.Context, id uint64, fio *string, phone *string) error
	GetRealty(ctx context.Context, id uint64) (*Realty, error)
	GaugeExists(ctx context.Context, id uint64) (bool, error)
	TariffExists(ctx context.Context, id uint64) (bool, error)
	AddMeterReading(ctx context.Context, realtyID uint64, req *PostReadingRequest) error
	GetMeterReadings(ctx context.Context, realtyID uint64, fromDate, toDate string) ([]MeterReading, error)
	GetInvoices(ctx context.Context, realtyID uint64, limit, offset int) ([]Invoice, int, error)
	GetTariffByDate(ctx context.Context, dateStr string) (*Tariff, error)
	BindGauge(ctx context.Context, realtyID uint64, gaugeID uint64) error
}

type ClickHouseRepository struct {
	db *sql.DB
}

func NewClickHouseRepository(db *sql.DB) *ClickHouseRepository {
	return &ClickHouseRepository{db: db}
}

func (r *ClickHouseRepository) GetUser(ctx context.Context, id uint64) (*User, error) {
	query := `SELECT id, fio, phone, phone_prev FROM dim_users FINAL WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)
	var u User
	err := row.Scan(&u.ID, &u.FIO, &u.Phone, &u.PhonePrev)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *ClickHouseRepository) UpdateUser(ctx context.Context, id uint64, fio *string, phone *string) error {
	u, err := r.GetUser(ctx, id)
	if err != nil {
		return err
	}
	if u == nil {
		return sql.ErrNoRows
	}

	newFIO := u.FIO
	if fio != nil {
		newFIO = *fio
	}

	newPhone := u.Phone
	newPhonePrev := u.PhonePrev
	if phone != nil && *phone != u.Phone {
		newPhonePrev = u.Phone
		newPhone = *phone
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	insertQuery := `INSERT INTO dim_users (id, fio, phone, phone_prev, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`
	_, err = r.db.ExecContext(ctx, insertQuery, u.ID, newFIO, newPhone, newPhonePrev, now, now)
	return err
}

func (r *ClickHouseRepository) GetRealty(ctx context.Context, id uint64) (*Realty, error) {
	query := `SELECT id, address, square, type, number, rooms, floor, porch FROM dim_realty WHERE id = ? LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, id)
	var re Realty
	err := row.Scan(&re.ID, &re.Address, &re.Square, &re.Type, &re.Number, &re.Rooms, &re.Floor, &re.Porch)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &re, nil
}

func (r *ClickHouseRepository) GaugeExists(ctx context.Context, id uint64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT count() FROM dim_gauges WHERE id = ?`, id).Scan(&count)
	return count > 0, err
}

func (r *ClickHouseRepository) TariffExists(ctx context.Context, id uint64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT count() FROM dim_tariffs FINAL WHERE id = ?`, id).Scan(&count)
	return count > 0, err
}

func (r *ClickHouseRepository) AddMeterReading(ctx context.Context, realtyID uint64, req *PostReadingRequest) error {
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	query := `INSERT INTO fact_meter_readings (realty_id, user_id, tariff_id, gauge_id, reading_date, day_kw, night_kw, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := r.db.ExecContext(ctx, query, realtyID, req.UserID, req.TariffID, req.GaugeID, req.ReadingDate, req.DayKW, req.NightKW, now)
	return err
}

func (r *ClickHouseRepository) GetMeterReadings(ctx context.Context, realtyID uint64, fromDate, toDate string) ([]MeterReading, error) {
	query := `
		SELECT reading_date, user_id, tariff_id, gauge_id, day_kw, night_kw
		FROM fact_meter_readings FINAL
		WHERE realty_id = ? AND reading_date >= ? AND reading_date <= ?
		ORDER BY reading_date ASC`
	rows, err := r.db.QueryContext(ctx, query, realtyID, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	readings := make([]MeterReading, 0)
	for rows.Next() {
		var mr MeterReading
		var d time.Time
		if err := rows.Scan(&d, &mr.UserID, &mr.TariffID, &mr.GaugeID, &mr.DayKW, &mr.NightKW); err != nil {
			return nil, err
		}
		mr.ReadingDate = d.Format("2006-01-02")
		readings = append(readings, mr)
	}
	return readings, nil
}

func (r *ClickHouseRepository) GetInvoices(ctx context.Context, realtyID uint64, limit, offset int) ([]Invoice, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, `SELECT count() FROM fact_invoice WHERE realty_id = ?`, realtyID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT tariff_id, gauge_id, day_kw, night_kw, price, deadline, created_at
		FROM fact_invoice
		WHERE realty_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`
	rows, err := r.db.QueryContext(ctx, query, realtyID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	invoices := make([]Invoice, 0)
	for rows.Next() {
		var inv Invoice
		var dl time.Time
		var ca time.Time
		if err := rows.Scan(&inv.TariffID, &inv.GaugeID, &inv.DayKW, &inv.NightKW, &inv.Price, &dl, &ca); err != nil {
			return nil, 0, err
		}
		inv.Deadline = dl.Format("2006-01-02")
		inv.CreatedAt = ca.Format("2006-01-02 15:04:05")
		invoices = append(invoices, inv)
	}
	return invoices, total, nil
}

func (r *ClickHouseRepository) GetTariffByDate(ctx context.Context, dateStr string) (*Tariff, error) {
	dt, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date")
	}
	dtStr := dt.Format("2006-01-02 00:00:00")

	query := `
		SELECT id, name, price_day_kwph, price_night_kwph, valid_from, valid_to
		FROM dim_tariffs FINAL
		WHERE valid_from <= ? AND valid_to > ?
		LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, dtStr, dtStr)
	var t Tariff
	var vf, vt time.Time
	err = row.Scan(&t.ID, &t.Name, &t.PriceDayKWPH, &t.PriceNightKWPH, &vf, &vt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.ValidFrom = vf.Format("2006-01-02 15:04:05")
	t.ValidTo = vt.Format("2006-01-02 15:04:05")
	return &t, nil
}

func (r *ClickHouseRepository) BindGauge(ctx context.Context, realtyID uint64, gaugeID uint64) error {
	queryActive := `
		SELECT gauge_id, version
		FROM fact_gauge_bindings FINAL
		WHERE realty_id = ?
		HAVING sum(sign) > 0
		LIMIT 1`
	row := r.db.QueryRowContext(ctx, queryActive, realtyID)

	var activeGaugeID uint64
	var activeVersion uint64
	hasActive := false
	err := row.Scan(&activeGaugeID, &activeVersion)
	if err == nil {
		hasActive = true
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")

	if hasActive {
		cancelQuery := `INSERT INTO fact_gauge_bindings (gauge_id, realty_id, created_at, sign, version) VALUES (?, ?, ?, -1, ?)`
		if _, err := r.db.ExecContext(ctx, cancelQuery, activeGaugeID, realtyID, now, activeVersion); err != nil {
			return err
		}
	}

	newVersion := uint64(1)
	if hasActive {
		newVersion = activeVersion + 1
	}

	insertQuery := `INSERT INTO fact_gauge_bindings (gauge_id, realty_id, created_at, sign, version) VALUES (?, ?, ?, 1, ?)`
	_, err = r.db.ExecContext(ctx, insertQuery, gaugeID, realtyID, now, newVersion)
	return err
}

// Handlers & Server Setup

type Server struct {
	repo DatabaseRepository
	log  *slog.Logger
}

type PostReadingRequest struct {
	UserID      uint64 `json:"user_id"`
	GaugeID     uint64 `json:"gauge_id"`
	TariffID    uint64 `json:"tariff_id"`
	ReadingDate string `json:"reading_date"`
	DayKW       uint32 `json:"day_kw"`
	NightKW     uint32 `json:"night_kw"`
}

type PatchUserRequest struct {
	FIO   *string `json:"fio"`
	Phone *string `json:"phone"`
}

type BindGaugeRequest struct {
	GaugeID uint64 `json:"gauge_id"`
}

func jsonError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/users/")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		jsonError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	ctx := r.Context()

	switch r.Method {
	case http.MethodGet:
		user, err := s.repo.GetUser(ctx, id)
		if err != nil {
			s.log.Error("failed to get user", "error", err)
			jsonError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if user == nil {
			jsonError(w, http.StatusNotFound, "Not found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(user)

	case http.MethodPatch:
		var req PatchUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			jsonError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.FIO == nil && req.Phone == nil {
			jsonError(w, http.StatusBadRequest, "invalid \"fio\" field")
			return
		}

		err := s.repo.UpdateUser(ctx, id, req.FIO, req.Phone)
		if errors.Is(err, sql.ErrNoRows) {
			jsonError(w, http.StatusNotFound, "Not found")
			return
		}
		if err != nil {
			s.log.Error("failed to update user", "error", err)
			jsonError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRealty(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/realty/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		jsonError(w, http.StatusBadRequest, "invalid realty id")
		return
	}

	realtyID, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || realtyID == 0 {
		jsonError(w, http.StatusBadRequest, "invalid realty id")
		return
	}

	ctx := r.Context()

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		realty, err := s.repo.GetRealty(ctx, realtyID)
		if err != nil {
			s.log.Error("failed to get realty", "error", err)
			jsonError(w, http.StatusInternalServerError, "internal server error")
			return
		}
		if realty == nil {
			jsonError(w, http.StatusNotFound, "Not found")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(realty)
		return
	}

	action := parts[1]
	switch action {
	case "readings":
		if r.Method == http.MethodPost {
			s.handlePostReading(w, r, realtyID)
		} else if r.Method == http.MethodGet {
			s.handleGetReadings(w, r, realtyID)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	case "invoices":
		if r.Method == http.MethodGet {
			s.handleGetInvoices(w, r, realtyID)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	case "gauges":
		if r.Method == http.MethodPost {
			s.handleBindGauge(w, r, realtyID)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	default:
		jsonError(w, http.StatusNotFound, "Not found")
	}
}

func (s *Server) handlePostReading(w http.ResponseWriter, r *http.Request, realtyID uint64) {
	ctx := r.Context()
	realty, err := s.repo.GetRealty(ctx, realtyID)
	if err != nil {
		s.log.Error("failed to check realty", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if realty == nil {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}

	var req PostReadingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.UserID == 0 {
		jsonError(w, http.StatusBadRequest, "invalid \"user_id\" field")
		return
	}
	if req.GaugeID == 0 {
		jsonError(w, http.StatusBadRequest, "invalid \"gauge_id\" field")
		return
	}
	if req.TariffID == 0 {
		jsonError(w, http.StatusBadRequest, "invalid \"tariff_id\" field")
		return
	}
	if req.ReadingDate == "" {
		jsonError(w, http.StatusBadRequest, "invalid \"reading_date\" field")
		return
	}
	if _, err := time.Parse("2006-01-02", req.ReadingDate); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid \"reading_date\" field")
		return
	}

	user, err := s.repo.GetUser(ctx, req.UserID)
	if err != nil || user == nil {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}
	gExists, err := s.repo.GaugeExists(ctx, req.GaugeID)
	if err != nil || !gExists {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}
	tExists, err := s.repo.TariffExists(ctx, req.TariffID)
	if err != nil || !tExists {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}

	if err := s.repo.AddMeterReading(ctx, realtyID, &req); err != nil {
		s.log.Error("failed to add reading", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleGetReadings(w http.ResponseWriter, r *http.Request, realtyID uint64) {
	ctx := r.Context()
	realty, err := s.repo.GetRealty(ctx, realtyID)
	if err != nil {
		s.log.Error("failed to check realty", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if realty == nil {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}

	from := r.URL.Query().Get("date_from")
	to := r.URL.Query().Get("date_to")

	if from == "" {
		jsonError(w, http.StatusBadRequest, "invalid \"date_from\" parameter")
		return
	}
	if to == "" {
		jsonError(w, http.StatusBadRequest, "invalid \"date_to\" parameter")
		return
	}
	if _, err := time.Parse("2006-01-02", from); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid \"date_from\" parameter")
		return
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid \"date_to\" parameter")
		return
	}

	readings, err := s.repo.GetMeterReadings(ctx, realtyID, from, to)
	if err != nil {
		s.log.Error("failed to get readings", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"readings": readings,
		"count":    len(readings),
	})
}

func (s *Server) handleGetInvoices(w http.ResponseWriter, r *http.Request, realtyID uint64) {
	ctx := r.Context()
	realty, err := s.repo.GetRealty(ctx, realtyID)
	if err != nil {
		s.log.Error("failed to check realty", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if realty == nil {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}

	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")

	limit := 10
	offset := 0
	var errP error

	if limitStr != "" {
		limit, errP = strconv.Atoi(limitStr)
		if errP != nil || limit < 0 {
			jsonError(w, http.StatusBadRequest, "invalid \"limit\" parameter")
			return
		}
	}
	if offsetStr != "" {
		offset, errP = strconv.Atoi(offsetStr)
		if errP != nil || offset < 0 {
			jsonError(w, http.StatusBadRequest, "invalid \"offset\" parameter")
			return
		}
	}

	invoices, total, err := s.repo.GetInvoices(ctx, realtyID, limit, offset)
	if err != nil {
		s.log.Error("failed to get invoices", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"invoices": invoices,
		"count":    total,
	})
}

func (s *Server) handleBindGauge(w http.ResponseWriter, r *http.Request, realtyID uint64) {
	ctx := r.Context()
	realty, err := s.repo.GetRealty(ctx, realtyID)
	if err != nil {
		s.log.Error("failed to check realty", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	if realty == nil {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}

	var req BindGaugeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GaugeID == 0 {
		jsonError(w, http.StatusBadRequest, "invalid \"gauge_id\" field")
		return
	}

	gExists, err := s.repo.GaugeExists(ctx, req.GaugeID)
	if err != nil || !gExists {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}

	if err := s.repo.BindGauge(ctx, realtyID, req.GaugeID); err != nil {
		s.log.Error("failed to bind gauge", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (s *Server) handleTariffs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		jsonError(w, http.StatusBadRequest, "invalid \"date\" parameter")
		return
	}

	tariff, err := s.repo.GetTariffByDate(r.Context(), dateStr)
	if err != nil {
		if strings.Contains(err.Error(), "invalid date") {
			jsonError(w, http.StatusBadRequest, "invalid \"date\" parameter")
			return
		}
		s.log.Error("failed to get tariff", "error", err)
		jsonError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if tariff == nil {
		jsonError(w, http.StatusNotFound, "Not found")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(tariff)
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	chHost := os.Getenv("CLICKHOUSE_HOST")
	if chHost == "" {
		chHost = "localhost"
	}
	chPort := os.Getenv("CLICKHOUSE_PORT")
	if chPort == "" {
		chPort = "9000"
	}
	chDB := os.Getenv("CLICKHOUSE_DATABASE")
	if chDB == "" {
		chDB = "energosbyt"
	}
	chUser := os.Getenv("CLICKHOUSE_USER")
	if chUser == "" {
		chUser = "default"
	}
	chPassword := os.Getenv("CLICKHOUSE_PASSWORD")

	dsn := fmt.Sprintf("clickhouse://%s:%s@%s:%s/%s?dial_timeout=10s&read_timeout=20s",
		chUser, chPassword, chHost, chPort, chDB)

	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		logger.Error("failed to connect to clickhouse", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Error("failed to ping clickhouse", "error", err)
		os.Exit(1)
	}

	repo := NewClickHouseRepository(db)
	srv := &Server{repo: repo, log: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("/ping", srv.handlePing)
	mux.HandleFunc("/users/", srv.handleUsers)
	mux.HandleFunc("/realty/", srv.handleRealty)
	mux.HandleFunc("/tariffs", srv.handleTariffs)

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}

	logger.Info("starting http server", "port", appPort)
	if err := http.ListenAndServe(":"+appPort, mux); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
