package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

var (
	// randSource источник псевдо случайных чисел.
	// Для повышения уникальности в качестве seed
	// используется текущее время в unix формате (в виде числа)
	randSource = rand.NewSource(time.Now().UnixNano())
	// randRange использует randSource для генерации случайных чисел
	randRange = rand.New(randSource)
)

// getTestParcel возвращает тестовую посылку
func getTestParcel() Parcel {
	return Parcel{
		Client:    1000,
		Status:    ParcelStatusRegistered,
		Address:   "test",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// TestAddGetDelete проверяет добавление, получение и удаление посылки
func TestAddGetDelete(t *testing.T) {
	// Prepare - подключение к БД
	db, err := sql.Open("sqlite", "tracker.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()
	// Создаём хранилище посылок (ParcelStore) с использованием подключения к БД (db)
	store := NewParcelStore(db)
	parcel := getTestParcel()

	// Add - добавляем новую посылку в БД
	id, err := store.Add(parcel)
	require.NoError(t, err, "failed to add parcel")
	require.Greater(t, id, 0, "parcel ID should be positive") // идентификатор должен быть больше нуля
	// Обновляем идентификатор добавленной посылки
	parcel.Number = id

	// Get - получаем только что добавленную посылку
	newParcel, err := store.Get(id) //
	require.NoError(t, err, "failed to get parcel")
	// Проверяем соответствие всех полей
	require.Equal(t, parcel.Number, newParcel.Number, "parcel number mismatch")
	require.Equal(t, parcel.Client, newParcel.Client, "client ID mismatch")
	require.Equal(t, parcel.Status, newParcel.Status, "status mismatch")
	require.Equal(t, parcel.Address, newParcel.Address, "address mismatch")
	require.Equal(t, parcel.CreatedAt, newParcel.CreatedAt, "created at mismatch")

	// Delete - Удаляем добавленную посылку
	err = store.Delete(id)
	require.NoError(t, err, "failed to delete parcel")
	// Проверяем, что посылку больше нельзя получить из БД
	_, err = store.Get(id)
	require.Error(t, err, "deleted parcel should not be retrievable")
	require.Equal(t, err, sql.ErrNoRows, "expected sql.ErrNoRows when getting deleted parcel")
}

// TestSetAddress проверяет обновление адреса
func TestSetAddress(t *testing.T) {
	// Prepare
	db, err := sql.Open("sqlite", "tracker.db")
	require.NoError(t, err, "failed to connect to database")
	defer db.Close()
	// Создаём хранилище посылок (ParcelStore) с использованием подключения к БД (db)
	store := NewParcelStore(db)
	parcel := getTestParcel()

	// Add - Добавляем новую посылку в БД
	id, err := store.Add(parcel)
	require.NoError(t, err, "failed to add parcel")
	require.Greater(t, id, 0, "parcel ID should be positive")
	// Обновляем идентификатор добавленной посылки
	parcel.Number = id

	// Set address - обновляем адрес
	newAddress := "new test address"
	err = store.SetAddress(id, newAddress)
	require.NoError(t, err, "failed to update parcel address")

	// Check - получаем добавленную посылку
	updatedParcel, err := store.Get(id)
	require.NoError(t, err, "failed to get updated parcel")
	// Проверяем, что адрес обновился
	require.Equal(t, newAddress, updatedParcel.Address,
		"address should be updated to '%s', but got '%s'",
		newAddress, updatedParcel.Address)
	// Проверяем, что остальные поля остались без изменений
	require.Equal(t, parcel.Client, updatedParcel.Client, "client ID should remain unchanged")
	require.Equal(t, parcel.Status, updatedParcel.Status, "status should remain unchanged")
	require.Equal(t, parcel.CreatedAt, updatedParcel.CreatedAt, "created at should remain unchanged")
}

// TestSetStatus проверяет обновление статуса
func TestSetStatus(t *testing.T) {
	// Prepare
	db, err := sql.Open("sqlite", "tracker.db")
	require.NoError(t, err, "failed to connect to database")
	defer db.Close()
	// Создаём хранилище посылок (ParcelStore) с использованием подключения к БД (db)
	store := NewParcelStore(db)
	parcel := getTestParcel()

	// Add - Добавляем новую посылку в БД
	id, err := store.Add(parcel)
	require.NoError(t, err, "failed to add parcel")
	require.Greater(t, id, 0, "parcel ID should be positive")
	// Обновляем идентификатор добавленной посылки
	parcel.Number = id

	// Set status - обновляем статус
	newStatus := ParcelStatusSent
	err = store.SetStatus(id, newStatus)
	require.NoError(t, err, "failed to update parcel status")

	// Check - получаем добавленную посылку
	updatedParcel, err := store.Get(id)
	require.NoError(t, err, "failed to get updated parcel")
	// Проверяем, что статус обновился
	require.Equal(t, newStatus, updatedParcel.Status,
		"status should be updated to '%s', but got '%s'",
		newStatus, updatedParcel.Status)
	// Проверяем, что остальные поля остались без изменений
	require.Equal(t, parcel.Client, updatedParcel.Client, "client ID should remain unchanged")
	require.Equal(t, parcel.Address, updatedParcel.Address, "address should remain unchanged")
	require.Equal(t, parcel.CreatedAt, updatedParcel.CreatedAt, "created at should remain unchanged")
}

// TestGetByClient проверяет получение посылок по идентификатору клиента
func TestGetByClient(t *testing.T) {
	// Prepare
	db, err := sql.Open("sqlite", "tracker.db")
	require.NoError(t, err, "failed to connect to database")
	defer db.Close()
	// Создаём хранилище посылок (ParcelStore) с использованием подключения к БД (db)
	store := NewParcelStore(db)
	parcels := []Parcel{
		getTestParcel(),
		getTestParcel(),
		getTestParcel(),
	}
	parcelMap := map[int]Parcel{}
	// Задаём всем посылкам один и тот же идентификатор клиента
	client := randRange.Intn(10_000_000)
	parcels[0].Client = client
	parcels[1].Client = client
	parcels[2].Client = client

	// Add - добавляем посылки в БД и сохраняем их ID
	for i := 0; i < len(parcels); i++ {
		id, err := store.Add(parcels[i])
		require.NoError(t, err, "failed to add parcel %d", i)
		require.Greater(t, id, 0, "parcel ID should be positive for parcel %d", i)
		// Обновляем идентификатор добавленной посылки
		parcels[i].Number = id
		// Сохраняем добавленную посылку в структуру map, чтобы её можно было легко достать по идентификатору посылки
		parcelMap[id] = parcels[i]
	}

	// Get by client - получаем список посылок по идентификатору клиента
	storedParcels, err := store.GetByClient(client)
	require.NoError(t, err, "failed to get parcels by client %d", client)
	// Проверяем, что количество полученных посылок совпадает с количеством добавленных
	require.Len(t, storedParcels, len(parcels),
		"number of retrieved parcels (%d) should match number of added parcels (%d)",
		len(storedParcels), len(parcels))

	// Check - проверяем корректность полученных данных
	for _, parcel := range storedParcels {
		// Проверяем, что посылка из storedParcels есть в parcelMap
		expectedParcel, exists := parcelMap[parcel.Number]
		require.True(t, exists, "received parcel with number %d not found in original data", parcel.Number)
		// Проверяем, что значения полей полученных посылок заполнены верно
		require.Equal(t, expectedParcel.Number, parcel.Number, "parcel number mismatch for parcel %d", parcel.Number)
		require.Equal(t, expectedParcel.Client, parcel.Client, "client ID mismatch for parcel %d", parcel.Number)
		require.Equal(t, expectedParcel.Status, parcel.Status, "status mismatch for parcel %d", parcel.Number)
		require.Equal(t, expectedParcel.Address, parcel.Address, "address mismatch for parcel %d", parcel.Number)
		require.Equal(t, expectedParcel.CreatedAt, parcel.CreatedAt, "created at mismatch for parcel %d", parcel.Number)
	}
}
