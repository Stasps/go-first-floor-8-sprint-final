package main

import (
	"database/sql"
	"fmt"
)

// ParcelStore — структура для работы с данными о посылках в БД.
// Содержит подключение к базе данных.
type ParcelStore struct {
	db *sql.DB
}

// NewParcelStore — конструктор для создания объекта ParcelStore.
// Принимает подключение к БД и возвращает готовый объект.
func NewParcelStore(db *sql.DB) ParcelStore {
	return ParcelStore{db: db}
}

// Add добавляет новую посылку в таблицу parcel и возвращает её идентификатор.
// Параметры:
//
//	p — структура Parcel с данными посылки (статус должен быть ParcelStatusRegistered).
//
// Возвращает:
//
//	int — ID новой записи (автоинкрементное поле number).
//	error — ошибка при выполнении операции или nil при успехе.
func (s ParcelStore) Add(p Parcel) (int, error) {
	// Выполняем SQL‑запрос INSERT для добавления новой посылки в таблицу parcel
	res, err := s.db.Exec(
		"INSERT INTO parcel (client, status, address, created_at) VALUES (?, ?, ?, ?)",
		p.Client,    // ID клиента
		p.Status,    // Статус посылки (например, "registered")
		p.Address,   // Адрес доставки
		p.CreatedAt, // Дата и время создания (формат RFC3339)
	)
	if err != nil {
		// Ошибка БД: проблемы с подключением, синтаксис запроса или нарушение ограничений (например, NOT NULL)
		return 0, fmt.Errorf("failed to insert parcel: %w", err)
	}
	// Получаем ID последней вставленной записи (автоинкрементное поле number в таблице parcel)
	id, err := res.LastInsertId() // int64 - тип, возвращаемый LastInsertId()
	if err != nil {
		// Редкий случай: драйвер БД не смог вернуть ID (например, из‑за ошибки протокола)
		return 0, fmt.Errorf("failed to get last insert ID: %w", err)
	}
	// Успех: возвращаем идентификатор последней добавленной записи
	return int(id), nil
}

// Get получает посылку из таблицы parcel по её номеру.
// Параметры:
//
//	number — номер посылки (соответствует полю number в таблице).
//
// Возвращает:
//
//	Parcel — заполненная структура с данными посылки или Parcel{} при ошибке.
//	error — sql.ErrNoRows, если посылка не найдена, или другая ошибка БД.
func (s ParcelStore) Get(number int) (Parcel, error) {
	// Проверяем входные параметры на корректность перед выполнением запроса.
	if number <= 0 {
		return Parcel{}, fmt.Errorf("invalid parcel number: %d", number)
	}
	// Выполняем SQL-запрос SELECT для получения одной строки по номеру посылки
	row := s.db.QueryRow(
		"SELECT number, client, status, address, created_at FROM parcel WHERE number = ?",
		number,
	)
	// Создаём пустую структуру для заполнения данными
	p := Parcel{}
	err := row.Scan(
		&p.Number,
		&p.Client,
		&p.Status,
		&p.Address,
		&p.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Ошибка: Посылка с указанным номером не найдена в БД
			return Parcel{}, sql.ErrNoRows
		}
		// Ошибка при чтении данных (например, несоответствие типов)
		return Parcel{}, fmt.Errorf("failed to scan parcel data: %w", err)
	}
	// Успех: возвращаем заполненную структуру с данными посылки
	return p, nil
}

// GetByClient получает все посылки для указанного клиента.
// Параметры:
//
//	client — ID клиента.
//
// Возвращает:
//
//	[]Parcel — срез посылок клиента (пустой, если посылок нет).
//	error — ошибка БД или nil.
func (s ParcelStore) GetByClient(client int) ([]Parcel, error) {
	// Выполняем SQL-запрос SELECT для получения всех строк по указанному клиенту
	rows, err := s.db.Query(
		"SELECT number, client, status, address, created_at FROM parcel WHERE client = ?",
		client,
	)
	if err != nil {
		// Ошибка при выполнении запроса (например, синтаксическая ошибка или проблема с подключением)
		return nil, fmt.Errorf("failed to query parcels for client %d: %w", client, err)
	}
	defer rows.Close() // Гарантированно закрываем курсор после завершения функции
	// Инициализируем пустой срез для хранения результатов
	var res []Parcel
	// Заполняем массив в переменной res по всем строкам, возвращённым запросом
	for rows.Next() {
		var temp Parcel // Временный объект для текущей строки
		err := rows.Scan(&temp.Number, &temp.Client, &temp.Status, &temp.Address, &temp.CreatedAt)
		if err != nil {
			// Ошибка при чтении данных конкретной строки (например, проблема с типом данных)
			return nil, fmt.Errorf("failed to scan parcel row: %w", err)
		}
		// Добавляем заполненный объект в результирующий срез
		res = append(res, temp)
	}
	// Проверяем, не возникла ли ошибка во время итерации по строкам
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}
	// Успех: возвращаем срез с посылками клиента (может быть пустым, если посылок нет)
	return res, nil
}

// SetStatus обновляет статус посылки в таблице parcel.
// Параметры:
//
//	number — номер посылки.
//	status — новый статус ("sent" или "delivered").
//
// Возвращает:
//
//	error — ошибка БД или nil при успешном обновлении.
func (s ParcelStore) SetStatus(number int, status string) error {
	// Проверяем входные параметры на корректность перед выполнением запроса.
	if number <= 0 {
		return fmt.Errorf("invalid parcel number: %d", number)
	}
	// Выполняем SQL-запрос UPDATE для изменения статуса посылки по её номеру
	result, err := s.db.Exec(
		"UPDATE parcel SET status = ? WHERE number = ?",
		status,
		number,
	)
	if err != nil {
		// Ошибка БД: проблемы с подключением, синтаксис запроса или блокировка таблицы
		return fmt.Errorf("failed to update parcel status for number %d: %w", number, err)
	}
	// Получаем количество строк, затронутых запросом: если 0 — посылка не найдена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Редкий случай: драйвер не смог вернуть статистику (например, ошибка протокола)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Ни одна строка не обновлена: посылка не найдена ИЛИ статус нельзя изменить (например, "delivered")
		return fmt.Errorf("parcel with number %d not found or cannot be updated (check current status)", number)
	}
	// Успех: статус посылки успешно изменён, запись существует
	return nil
}

// SetAddress обновляет адрес доставки посылки.
// Менять адрес можно только если статус посылки — "registered".
// Параметры:
//
//	number — номер посылки.
//	address — новый адрес доставки.
//
// Возвращает:
//
//	error — ошибка БД, бизнес‑ошибка (если статус не "registered") или nil.
func (s ParcelStore) SetAddress(number int, address string) error {
	// Проверяем входные параметры на корректность перед выполнением запроса.
	if number <= 0 {
		return fmt.Errorf("invalid parcel number: %d", number)
	}
	// Выполняем SQL-запрос UPDATE для изменения адреса доставки посылки по её номеру
	result, err := s.db.Exec(
		"UPDATE parcel SET address = ? WHERE number = ? AND status = ?",
		address,
		number,
		ParcelStatusRegistered, // Менять адрес можно только если статус посылки — "registered".
	)
	if err != nil {
		// Ошибка БД: проблемы с подключением, синтаксис запроса или блокировка таблицы
		return fmt.Errorf("failed to update parcel address for number %d: %w", number, err)
	}
	// Получаем количество строк, затронутых запросом: если 0 — посылка не найдена
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Редкий случай: драйвер не смог вернуть статистику (например, ошибка протокола)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Ни одна строка не изменена: посылка не найдена ИЛИ статус не "registered"
		return fmt.Errorf("parcel with number %d not found or has invalid status (must be '%s')", number, ParcelStatusRegistered)
	}
	// Успех: адрес доставки успешно изменён для существующей посылки в статусе "registered"
	return nil
}

// Delete удаляет посылку из таблицы parcel.
// Удалять посылку можно только если её статус — "registered".
// Параметры:
//
//	number — номер посылки (соответствует полю number в таблице).
//
// Возвращает:
//
//	error — ошибка БД, бизнес‑ошибка (если статус не "registered" или посылка не найдена) или nil при успешном удалении.
func (s ParcelStore) Delete(number int) error {
	// Проверяем входные параметры на корректность перед выполнением запроса.
	if number <= 0 {
		return fmt.Errorf("invalid parcel number: %d", number)
	}
	// Выполняем SQL‑запрос DELETE: удаляем посылку с указанным номером,
	result, err := s.db.Exec(
		"DELETE FROM parcel WHERE number = ? AND status = ?",
		number,
		ParcelStatusRegistered, // Удалять можно только посылки со статусом "registered"
	)
	if err != nil {
		// Ошибка БД: проблемы с подключением, синтаксис запроса или блокировка таблицы
		return fmt.Errorf("failed to delete parcel for number %d: %w", number, err)
	}
	// Получаем количество строк, затронутых запросом: если 0 — посылка не найдена ИЛИ статус не "registered"
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		// Редкий случай: драйвер не смог вернуть статистику (например, ошибка протокола)
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		// Ни одна строка не удалена: посылка не найдена ИЛИ статус не "registered"
		return fmt.Errorf("parcel with number %d not found or has invalid status (must be '%s')", number, ParcelStatusRegistered)
	}
	// Успех: запись о посылке удалена из таблицы
	return nil
}
