package main

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type GeneratorRepository interface {
	GenerateRealty(path string) error
	GenerateGauges(path string) error
	GenerateTariffs(path string) error
	GenerateUsers(path string) error
	GenerateGaugeBindings(path string) error
	GenerateMeterReadings(path string) error
	GenerateInvoices(path string) error
}

type CSVDataRepository struct {
	rng *rand.Rand
	log *slog.Logger
}

func NewCSVDataRepository(seed int64, logger *slog.Logger) *CSVDataRepository {
	return &CSVDataRepository{
		rng: rand.New(rand.NewSource(seed)),
		log: logger,
	}
}

func (r *CSVDataRepository) GenerateRealty(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	types := []string{"flat", "nrcp"}
	streets := []string{"Пушкина", "Ленина", "Гагарина", "Мира", "Советская"}
	nowStr := "2023-10-01 00:00:00"

	for i := 1; i <= 1000; i++ {
		t := types[r.rng.Intn(len(types))]
		street := streets[r.rng.Intn(len(streets))]
		house := r.rng.Intn(50) + 1
		addr := fmt.Sprintf("г. Москва, ул. %s, д. %d", street, house)
		sq := strconv.Itoa(r.rng.Intn(80) + 20)
		num := ""
		rooms := "0"
		if t == "flat" {
			num = strconv.Itoa(r.rng.Intn(200) + 1)
			rooms = strconv.Itoa(r.rng.Intn(4) + 1)
		}
		floor := strconv.Itoa(r.rng.Intn(16) + 1)
		porch := strconv.Itoa(r.rng.Intn(4) + 1)

		if err := w.Write([]string{
			strconv.Itoa(i),
			addr,
			sq,
			t,
			num,
			rooms,
			floor,
			porch,
			nowStr,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *CSVDataRepository) GenerateGauges(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	models := []string{"Меркурий 230", "Нева MT 114", "Энергомера CE102M"}
	countries := []string{"Россия", "Беларусь"}
	nowStr := "2023-10-01 00:00:00"

	for i := 1; i <= 1200; i++ {
		m := models[r.rng.Intn(len(models))]
		c := countries[r.rng.Intn(len(countries))]
		if err := w.Write([]string{
			strconv.Itoa(i),
			m,
			c,
			nowStr,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *CSVDataRepository) GenerateTariffs(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	rows := [][]string{
		{"1", "День-Ночь v1", "550", "275", "2023-01-01 00:00:00", "2024-07-01 00:00:00", "2023-01-01 00:00:00", "2023-01-01 00:00:00"},
		{"1", "День-Ночь v2", "620", "310", "2024-07-01 00:00:00", "2099-01-01 00:00:00", "2024-07-01 00:00:00", "2024-07-01 00:00:00"},
	}

	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func (r *CSVDataRepository) GenerateUsers(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	nowStr := "2023-10-01 00:00:00"

	for i := 1; i <= 1000; i++ {
		fio := fmt.Sprintf("Пользователь %d", i)
		phone := fmt.Sprintf("+7900%07d", i)
		if err := w.Write([]string{
			strconv.Itoa(i),
			fio,
			phone,
			"",
			nowStr,
			nowStr,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (r *CSVDataRepository) GenerateGaugeBindings(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	installDate := "2023-10-01 00:00:00"
	replaceDate := "2024-11-01 00:00:00"
	replacementCount := 200
	replacementGaugeID := 1001

	for realtyID := 1; realtyID <= 1000; realtyID++ {
		gaugeID := realtyID
		if err := w.Write([]string{
			strconv.Itoa(gaugeID),
			strconv.Itoa(realtyID),
			installDate,
			"1",
			"1",
		}); err != nil {
			return err
		}

		if realtyID <= replacementCount {
			if err := w.Write([]string{
				strconv.Itoa(gaugeID),
				strconv.Itoa(realtyID),
				replaceDate,
				"-1",
				"1",
			}); err != nil {
				return err
			}
			if err := w.Write([]string{
				strconv.Itoa(replacementGaugeID),
				strconv.Itoa(realtyID),
				replaceDate,
				"1",
				"2",
			}); err != nil {
				return err
			}
			replacementGaugeID++
		}
	}
	return nil
}

func (r *CSVDataRepository) GenerateMeterReadings(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	startDate := time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)

	replacementGaugeMap := make(map[int]int)
	for i := 1; i <= 200; i++ {
		replacementGaugeMap[i] = 1000 + i
	}

	for curr := startDate; !curr.After(endDate); curr = curr.AddDate(0, 0, 1) {
		dateStr := curr.Format("2006-01-02")
		createdStr := curr.Format("2006-01-02") + " 08:00:00"

		for realtyID := 1; realtyID <= 1000; realtyID++ {
			userID := realtyID
			tariffID := 1
			gaugeID := realtyID
			if realtyID <= 200 && curr.After(time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)) {
				gaugeID = replacementGaugeMap[realtyID]
			}

			dayKw := 1000 + r.rng.Intn(500)
			nightKw := 200 + r.rng.Intn(100)

			if err := w.Write([]string{
				strconv.Itoa(realtyID),
				strconv.Itoa(userID),
				strconv.Itoa(tariffID),
				strconv.Itoa(gaugeID),
				dateStr,
				strconv.Itoa(dayKw),
				strconv.Itoa(nightKw),
				createdStr,
			}); err != nil {
				return err
			}

			if r.rng.Float32() < 0.03 {
				correctedCreatedStr := curr.Format("2006-01-02") + " 18:00:00"
				if err := w.Write([]string{
					strconv.Itoa(realtyID),
					strconv.Itoa(userID),
					strconv.Itoa(tariffID),
					strconv.Itoa(gaugeID),
					dateStr,
					strconv.Itoa(dayKw + 5),
					strconv.Itoa(nightKw + 2),
					correctedCreatedStr,
				}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *CSVDataRepository) GenerateInvoices(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	w := csv.NewWriter(file)
	defer w.Flush()

	startYear, startMonth := 2023, 11

	for m := range 36 {
		dt := time.Date(startYear, time.Month(startMonth+m), 1, 0, 0, 0, 0, time.UTC)
		deadlineDt := time.Date(dt.Year(), dt.Month()+1, 10, 0, 0, 0, 0, time.UTC)
		createdDt := time.Date(dt.Year(), dt.Month()+1, 1, 10, 0, 0, 0, time.UTC)

		deadlineStr := deadlineDt.Format("2006-01-02")
		createdStr := createdDt.Format("2006-01-02 15:04:05")

		for realtyID := 1; realtyID <= 1000; realtyID++ {
			userID := realtyID
			tariffID := 1
			gaugeID := realtyID
			if realtyID <= 200 && dt.After(time.Date(2024, 11, 1, 0, 0, 0, 0, time.UTC)) {
				gaugeID = 1000 + realtyID
			}

			dayKw := 150 + r.rng.Intn(50)
			nightKw := 50 + r.rng.Intn(20)

			priceDay := 550
			priceNight := 275
			if dt.After(time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)) {
				priceDay = 620
				priceNight = 310
			}
			totalPrice := (dayKw * priceDay) + (nightKw * priceNight)

			if err := w.Write([]string{
				strconv.Itoa(realtyID),
				strconv.Itoa(userID),
				strconv.Itoa(tariffID),
				strconv.Itoa(gaugeID),
				strconv.Itoa(dayKw),
				strconv.Itoa(nightKw),
				strconv.Itoa(totalPrice),
				deadlineStr,
				createdStr,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	outDir := "data"

	if err := os.MkdirAll(outDir, 0755); err != nil {
		logger.Error("failed to create data output directory", "error", err)
		os.Exit(1)
	}

	repo := NewCSVDataRepository(42, logger)

	tasks := []struct {
		name string
		file string
		fn   func(string) error
	}{
		{"dim_realty", filepath.Join(outDir, "dim_realty.csv"), repo.GenerateRealty},
		{"dim_gauges", filepath.Join(outDir, "dim_gauges.csv"), repo.GenerateGauges},
		{"dim_tariffs", filepath.Join(outDir, "dim_tariffs.csv"), repo.GenerateTariffs},
		{"dim_users", filepath.Join(outDir, "dim_users.csv"), repo.GenerateUsers},
		{"fact_gauge_bindings", filepath.Join(outDir, "fact_gauge_bindings.csv"), repo.GenerateGaugeBindings},
		{"fact_meter_readings", filepath.Join(outDir, "fact_meter_readings.csv"), repo.GenerateMeterReadings},
		{"fact_invoice", filepath.Join(outDir, "fact_invoice.csv"), repo.GenerateInvoices},
	}

	for _, task := range tasks {
		logger.Info("generating dataset", "table", task.name, "file", task.file)
		if err := task.fn(task.file); err != nil {
			logger.Error("failed to generate dataset", "table", task.name, "error", err)
			os.Exit(1)
		}
	}

	logger.Info("dataset generation completed successfully")
}
