# ЭнергоСбыт — Лабораторная работа №1 (SCD в ClickHouse)

Проект представляет собой решение лабораторной работы №1 по моделированию данных компании «ЭнергоСбыт» по схеме «звезда» в ClickHouse и реализации HTTP-сервиса на Go с применением концепций SCD 1, SCD 2 и SCD 3.

## Обоснование выбора ключей сортировки (ORDER BY)

- **`dim_realty`**: `ORDER BY id` — быстрый точечный поиск объекта по идентификатору `id`.
- **`dim_gauges`**: `ORDER BY id` — прямой доступ к информации о счётчике.
- **`dim_tariffs`**: `ORDER BY (id, valid_from)` — поддержка SCD 2 (версионирование по временным интервалам) и выборок тарифа на произвольную дату.
- **`dim_users`**: `ORDER BY id` — замена/дедупликация в `ReplacingMergeTree` по уникальному идентификатору собственника.
- **`fact_gauge_bindings`**: `ORDER BY (realty_id, gauge_id)` — быстрый поиск текущих и исторических привязок счётчика к объекту в `VersionedCollapsingMergeTree`.
- **`fact_meter_readings`**: `ORDER BY (realty_id, reading_date)` — дедупликация повторных показаний за одну дату в `ReplacingMergeTree` по объекту.
- **`fact_invoice`**: `ORDER BY (realty_id, created_at)` — эффективная выборка счетов объекта с сортировкой по дате выставления.

## Запуск и использование

### Команды Makefile

- `make run` — Поднимает ClickHouse и HTTP-сервис через `docker compose`.
- `make stop` — Останавливает контейнеры.
- `make gen` — Генерирует CSV датасеты в папку `data/`.
- `make load` — Применяет Goose миграции и загружает CSV данные в ClickHouse.
- `make setup` — Создаёт схему, генерирует и загружает данные.
- `make teardown` — Очищает базу данных `energosbyt`.
- `make clean` — Останавливает сервисы и удаляет локальные Docker volumes.

## SCD сценарии и их проверка

### 1. Собственник сменил фамилию (SCD 1)

**Описание:** Фамилия перезаписывается напрямую, старое значение не сохраняется.

**Запрос:**
```http
PATCH /users/1 HTTP/1.1
Content-Type: application/json

{
    "fio": "Петрова Анна Сергеевна"
}
```

**Проверка через SQL:**
```sql
SELECT id, fio, phone, phone_prev FROM energosbyt.dim_users FINAL WHERE id = 1;
```
*Результат:* Поле `fio` равно "Петрова Анна Сергеевна", предыдущее значение отсутствует в базе.

---

### 2. Собственник сменил телефон дважды (SCD 3)

**Описание:** При обновлении телефона старый номер записывается в `phone_prev`, а новый — в `phone`.

**Запросы:**
```http
PATCH /users/1 HTTP/1.1
Content-Type: application/json

{
    "phone": "+79991112233"
}
```
затем:
```http
PATCH /users/1 HTTP/1.1
Content-Type: application/json

{
    "phone": "+79994445566"
}
```

**Проверка через SQL:**
```sql
SELECT id, phone, phone_prev FROM energosbyt.dim_users FINAL WHERE id = 1;
```
*Результат:* `phone` = "+79994445566", `phone_prev` = "+79991112233".

---

### 3. Изменился тариф с 1 ноября (SCD 2)

**Описание:** Тарифы имеют версии с интервалами действия `valid_from` и `valid_to`.

**Проверка через HTTP:**
- `GET /tariffs?date=2024-05-01` возвращает `price_day_kwph`: 550.
- `GET /tariffs?date=2024-08-01` возвращает `price_day_kwph`: 620.

**Проверка через SQL:**
```sql
SELECT id, name, price_day_kwph, valid_from, valid_to
FROM energosbyt.dim_tariffs FINAL
WHERE id = 1;
```
*Результат:* Возвращаются две строки тарифа с незавершённым и завершённым интервалом действия.

---

### 4. Пересчёт счёта за старый месяц (SCD 2)

**Описание:** Выставленный ранее счёт сохраняет цену на момент его формирования по тарифу, действовавшему на ту дату.

**Проверка через SQL:**
```sql
SELECT realty_id, price, created_at FROM energosbyt.fact_invoice WHERE realty_id = 1 ORDER BY created_at ASC LIMIT 1;
```

---

### 5. Корректировка показания (`ReplacingMergeTree`)

**Описание:** Повторная передача показаний за ту же дату добавляет новую запись с более поздним `created_at`. При выполнении запроса с `FINAL` выдаётся только актуальная запись.

**Запрос:**
```http
POST /realty/1/readings HTTP/1.1
Content-Type: application/json

{
    "user_id": 1,
    "gauge_id": 1,
    "tariff_id": 1,
    "reading_date": "2026-10-01",
    "day_kw": 1999,
    "night_kw": 500
}
```

**Проверка через SQL:**
```sql
SELECT * FROM energosbyt.fact_meter_readings FINAL WHERE realty_id = 1 AND reading_date = '2026-10-01';
```

---

### 6. Замена счётчика (`VersionedCollapsingMergeTree`)

**Описание:** При замене счётчика создаётся аннулирующая запись (`sign = -1`) для текущей активной привязки и новая запись установки (`sign = 1, version = v + 1`).

**Запрос:**
```http
POST /realty/1/gauges HTTP/1.1
Content-Type: application/json

{
    "gauge_id": 1005
}
```

**Проверка через SQL:**
```sql
SELECT gauge_id, realty_id, sign, version FROM energosbyt.fact_gauge_bindings FINAL WHERE realty_id = 1;
```
*Результат:* Возвращается одна строка с активным счётчиком `1005` и `sign = 1`.

---

### 7. Ошибочная строка с лишним нулём (DELETE)

**Описание:** Удаление конкретной ошибочной записи через мутацию ClickHouse.

**Запрос SQL:**
```sql
ALTER TABLE energosbyt.fact_meter_readings DELETE WHERE day_kw > 10000;
```

**Проверка через SQL:**
```sql
SELECT count() FROM energosbyt.fact_meter_readings WHERE day_kw > 10000;
```
*Результат:* `0`.
