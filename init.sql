-- Создание таблицы пользователей
CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY,
                                     email TEXT UNIQUE NOT NULL,
                                     password TEXT NOT NULL,
                                     role TEXT NOT NULL
);

-- Создание таблицы ПВЗ
CREATE TABLE IF NOT EXISTS pvz (
                                   id UUID PRIMARY KEY,
                                   registration_date TIMESTAMP NOT NULL,
                                   city TEXT NOT NULL
);

-- Создание таблицы приёмок
CREATE TABLE IF NOT EXISTS receptions (
                                          id UUID PRIMARY KEY,
                                          date_time TIMESTAMP NOT NULL,
                                          pvz_id UUID NOT NULL,
                                          status TEXT NOT NULL,
                                          FOREIGN KEY (pvz_id) REFERENCES pvz(id)
    );

-- Создание таблицы товаров
CREATE TABLE IF NOT EXISTS products (
                                        id UUID PRIMARY KEY,
                                        date_time TIMESTAMP NOT NULL,
                                        type TEXT NOT NULL,
                                        reception_id UUID NOT NULL,
                                        FOREIGN KEY (reception_id) REFERENCES receptions(id)
    );