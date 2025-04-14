
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS receptions;
DROP TABLE IF EXISTS pvz;
DROP TABLE IF EXISTS allowed_cities;
DROP TABLE IF EXISTS product_types;
DROP TABLE IF EXISTS users;

-- Таблица пользователей
CREATE TABLE IF NOT EXISTS users (
                                     id UUID PRIMARY KEY,
                                     email TEXT UNIQUE NOT NULL,
                                     password TEXT NOT NULL,
                                     role TEXT NOT NULL -- 'employee' or 'moderator'
);

-- Таблица разрешенных городов
CREATE TABLE IF NOT EXISTS allowed_cities (
                                              id SERIAL PRIMARY KEY,
                                              name TEXT UNIQUE NOT NULL
);

-- Таблица для типов товаров
CREATE TABLE IF NOT EXISTS product_types (
                                             id SERIAL PRIMARY KEY,
                                             name TEXT UNIQUE NOT NULL -- Название типа товара (электроника, одежда и т.д.)
);

-- Таблица ПВЗ
CREATE TABLE IF NOT EXISTS pvz (
                                   id UUID PRIMARY KEY,
                                   registration_date TIMESTAMP NOT NULL,
                                   city_id INT NOT NULL,       -- Foreign key to allowed_cities
                                   FOREIGN KEY (city_id) REFERENCES allowed_cities(id)
    );

-- Таблица приёмок
CREATE TABLE IF NOT EXISTS receptions (
                                          id UUID PRIMARY KEY,
                                          date_time TIMESTAMP NOT NULL,
                                          pvz_id UUID NOT NULL,
                                          status TEXT NOT NULL, -- 'in_progress' or 'close'
                                          FOREIGN KEY (pvz_id) REFERENCES pvz(id)
    );

-- Таблица товаров
CREATE TABLE IF NOT EXISTS products (
                                        id UUID PRIMARY KEY,
                                        date_time TIMESTAMP NOT NULL,
                                        reception_id UUID NOT NULL,
                                        type_id INT NOT NULL, -- Foreign key to product_types
                                        FOREIGN KEY (reception_id) REFERENCES receptions(id),
    FOREIGN KEY (type_id) REFERENCES product_types(id)
    );

-- Добавление начальных разрешенных городов (с проверкой на существование)
INSERT INTO allowed_cities (name)
SELECT 'Москва' WHERE NOT EXISTS (SELECT 1 FROM allowed_cities WHERE name = 'Москва');

INSERT INTO allowed_cities (name)
SELECT 'Санкт-Петербург' WHERE NOT EXISTS (SELECT 1 FROM allowed_cities WHERE name = 'Санкт-Петербург');

INSERT INTO allowed_cities (name)
SELECT 'Казань' WHERE NOT EXISTS (SELECT 1 FROM allowed_cities WHERE name = 'Казань');

-- Добавление начальных типов товаров (с проверкой на существование)
INSERT INTO product_types (name)
SELECT 'электроника' WHERE NOT EXISTS (SELECT 1 FROM product_types WHERE name = 'электроника');

INSERT INTO product_types (name)
SELECT 'одежда' WHERE NOT EXISTS (SELECT 1 FROM product_types WHERE name = 'одежда');

INSERT INTO product_types (name)
SELECT 'обувь' WHERE NOT EXISTS (SELECT 1 FROM product_types WHERE name = 'обувь');