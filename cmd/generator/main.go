package main

import (
	"encoding/csv"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	rand.Seed(42)
	outDir := "data"
	if err := os.MkdirAll(outDir, 0755); err != nil {
		panic(err)
	}

	fmt.Println("Generating dim_realty...")
	generateRealty(filepath.Join(outDir, "dim_realty.csv"))

	fmt.Println("Generating dim_gauges...")
	generateGauges(filepath.Join(outDir, "dim_gauges.csv"))

	fmt.Println("Generating dim_tariffs...")
	generateTariffs(filepath.Join(outDir, "dim_tariffs.csv"))

	fmt.Println("Generating dim_users...")
	generateUsers(filepath.Join(outDir, "dim_users.csv"))

	fmt.Println("Generating fact_gauge_bindings...")
	generateGaugeBindings(filepath.Join(outDir, "fact_gauge_bindings.csv"))

	fmt.Println("Generating fact_meter_readings...")
	generateMeterReadings(filepath.Join(outDir, "fact_meter_readings.csv"))

	fmt.Println("Generating fact_invoice...")
	generateInvoices(filepath.Join(outDir, "fact_invoice.csv"))

	fmt.Println("Generation complete!")
}

func generateRealty(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()

	types := []string{"flat", "nrcp"}
	streets := []string{"Пушкина", "Ленина", "Гагарина", "Мира", "Советская"}

	nowStr := "2023-10-01 00:00:00"

	for i := 1; i <= 1000; i++ {
		t := types[rand.Intn(len(types))]
		street := streets[rand.Intn(len(streets))]
		house := rand.Intn(50) + 1
		addr := fmt.Sprintf("г. Москва, ул. %s, д. %d", street, house)
		sq := strconv.Itoa(rand.Intn(80) + 20)
		num := ""
		rooms := "0"
		if t == "flat" {
			num = strconv.Itoa(rand.Intn(200) + 1)
			rooms = strconv.Itoa(rand.Intn(4) + 1)
		}
		floor := strconv.Itoa(rand.Intn(16) + 1)
		porch := strconv.Itoa(rand.Intn(4) + 1)

		w.Write([]string{
			strconv.Itoa(i),
			addr,
			sq,
			t,
			num,
			rooms,
			floor,
			porch,
			nowStr,
		})
	}
}

func generateGauges(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()

	models := []string{"Меркурий 230", "Нева MT 114", "Энергомера CE102M"}
	countries := []string{"Россия", "Беларусь"}
	nowStr := "2023-10-01 00:00:00"

	for i := 1; i <= 1200; i++ {
		m := models[rand.Intn(len(models))]
		c := countries[rand.Intn(len(countries))]
		w.Write([]string{
			strconv.Itoa(i),
			m,
			c,
			nowStr,
		})
	}
}

func generateTariffs(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()

	w.Write([]string{"1", "День-Ночь v1", "550", "275", "2023-01-01 00:00:00", "2024-07-01 00:00:00", "2023-01-01 00:00:00", "2023-01-01 00:00:00"})
	w.Write([]string{"1", "День-Ночь v2", "620", "310", "2024-07-01 00:00:00", "2099-01-01 00:00:00", "2024-07-01 00:00:00", "2024-07-01 00:00:00"})
}

func generateUsers(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()

	nowStr := "2023-10-01 00:00:00"

	for i := 1; i <= 1000; i++ {
		fio := fmt.Sprintf("Пользователь %d", i)
		phone := fmt.Sprintf("+7900%07d", i)
		w.Write([]string{
			strconv.Itoa(i),
			fio,
			phone,
			"",
			nowStr,
			nowStr,
		})
	}
}

func generateGaugeBindings(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
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
		w.Write([]string{
			strconv.Itoa(gaugeID),
			strconv.Itoa(realtyID),
			installDate,
			"1",
			"1",
		})

		if realtyID <= replacementCount {
			w.Write([]string{
				strconv.Itoa(gaugeID),
				strconv.Itoa(realtyID),
				replaceDate,
				"-1",
				"1",
			})
			w.Write([]string{
				strconv.Itoa(replacementGaugeID),
				strconv.Itoa(realtyID),
				replaceDate,
				"1",
				"2",
			})
			replacementGaugeID++
		}
	}
}

func generateMeterReadings(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
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

			dayKw := 1000 + rand.Intn(500)
			nightKw := 200 + rand.Intn(100)

			w.Write([]string{
				strconv.Itoa(realtyID),
				strconv.Itoa(userID),
				strconv.Itoa(tariffID),
				strconv.Itoa(gaugeID),
				dateStr,
				strconv.Itoa(dayKw),
				strconv.Itoa(nightKw),
				createdStr,
			})

			if rand.Float32() < 0.03 {
				correctedCreatedStr := curr.Format("2006-01-02") + " 18:00:00"
				w.Write([]string{
					strconv.Itoa(realtyID),
					strconv.Itoa(userID),
					strconv.Itoa(tariffID),
					strconv.Itoa(gaugeID),
					dateStr,
					strconv.Itoa(dayKw + 5),
					strconv.Itoa(nightKw + 2),
					correctedCreatedStr,
				})
			}
		}
	}
}

func generateInvoices(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	w := csv.NewWriter(file)
	defer w.Flush()

	startYear, startMonth := 2023, 11

	for m := 0; m < 36; m++ {
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

			dayKw := 150 + rand.Intn(50)
			nightKw := 50 + rand.Intn(20)

			priceDay := 550
			priceNight := 275
			if dt.After(time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC)) {
				priceDay = 620
				priceNight = 310
			}
			totalPrice := (dayKw * priceDay) + (nightKw * priceNight)

			w.Write([]string{
				strconv.Itoa(realtyID),
				strconv.Itoa(userID),
				strconv.Itoa(tariffID),
				strconv.Itoa(gaugeID),
				strconv.Itoa(dayKw),
				strconv.Itoa(nightKw),
				strconv.Itoa(totalPrice),
				deadlineStr,
				createdStr,
			})
		}
	}
}
