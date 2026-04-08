CREATE
    OR REPLACE FUNCTION lower_ru(text) RETURNS text AS
$$
SELECT translate($1,
                 'АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ',
                 'абвгдеёжзийклмнопрстуфхцчшщъыьэюя');
$$
    LANGUAGE SQL IMMUTABLE;

INSERT INTO "statuses" ("statusId", "title", "alias")
VALUES (1, 'Опубликован', 'enabled');
INSERT INTO "statuses" ("statusId", "title", "alias")
VALUES (2, 'Не опубликован', 'disabled');
INSERT INTO "statuses" ("statusId", "title", "alias")
VALUES (3, 'Удален', 'deleted');

-- password is 12345
INSERT INTO "users" ("login", "password", "statusId")
VALUES ('admin', '$2y$14$4IpqlaJ2Rvfgs.wb8f6lPODVLb/Ygl6zw1ZCUKz5CuT6WB6CV44AG', 1);

INSERT INTO "vfsFolders" ("parentFolderId", title, "isFavorite", "createdAt", "statusId")
VALUES (null, 'root', false, now(), 1);

-- Корневые категории
INSERT INTO "categories" ("title", "statusId")
VALUES ('Верх тела', 1),
       ('Низ тела', 1),
       ('Кор', 1);

-- Подкатегории верха (parentCategoryId = 1)
INSERT INTO "categories" ("title", "parentCategoryId", "statusId")
VALUES ('Грудь', 1, 1),
       ('Спина', 1, 1),
       ('Плечи', 1, 1),
       ('Руки', 1, 1);

-- Подкатегории низа (parentCategoryId = 2)
INSERT INTO "categories" ("title", "parentCategoryId", "statusId")
VALUES ('Квадрицепсы', 2, 1),
       ('Бицепс бедра', 2, 1),
       ('Ягодицы', 2, 1),
       ('Икры', 2, 1);

-- Подкатегории кора (parentCategoryId = 3)
INSERT INTO "categories" ("title", "parentCategoryId", "statusId")
VALUES ('Пресс', 3, 1),
       ('Косые мышцы', 3, 1),
       ('Поясница', 3, 1);

-- Упражнения (Грудь)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Жим лёжа', 4, 1),
       ('Жим гантелей лёжа', 4, 1),
       ('Разводка гантелей', 4, 1),
       ('Жим на наклонной скамье', 4, 1),
       ('Кроссовер', 4, 1),
       ('Отжимания', 4, 1);

-- Упражнения (Спина)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Подтягивания', 5, 1),
       ('Тяга верхнего блока', 5, 1),
       ('Тяга штанги в наклоне', 5, 1),
       ('Тяга гантели в наклоне', 5, 1),
       ('Гиперэкстензия', 5, 1),
       ('Тяга горизонтального блока', 5, 1);

-- Упражнения (Плечи)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Жим штанги стоя', 6, 1),
       ('Жим гантелей сидя', 6, 1),
       ('Разведение гантелей в стороны', 6, 1),
       ('Разведение в наклоне', 6, 1),
       ('Тяга к подбородку', 6, 1);

-- Упражнения (Руки)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Сгибание на бицепс', 7, 1),
       ('Сгибание гантелей', 7, 1),
       ('Молотки', 7, 1),
       ('Французский жим', 7, 1),
       ('Разгибание на блоке', 7, 1);

-- Упражнения (Квадрицепсы)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Приседания со штангой', 8, 1),
       ('Жим ногами', 8, 1),
       ('Разгибание ног в тренажёре', 8, 1),
       ('Фронтальные приседания', 8, 1);

-- Упражнения (Бицепс бедра)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Становая тяга на прямых ногах', 9, 1),
       ('Сгибание ног лёжа', 9, 1),
       ('Румынская тяга', 9, 1);

-- Упражнения (Ягодицы)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Ягодичный мост', 10, 1),
       ('Гиперэкстензия с акцентом на ягодицы', 10, 1),
       ('Выпады', 10, 1),
       ('Хип-траст', 10, 1);

-- Упражнения (Икры)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Подъёмы на носки стоя', 11, 1),
       ('Подъёмы на носки сидя', 11, 1),
       ('Жим носками в тренажёре', 11, 1);

-- Упражнения (Пресс)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Скручивания', 12, 1),
       ('Подъём ног в висе', 12, 1),
       ('Планка', 12, 1),
       ('Обратные скручивания', 12, 1);

-- Упражнения (Косые мышцы)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Скручивания с поворотом', 13, 1),
       ('Русские повороты', 13, 1),
       ('Боковая планка', 13, 1);

-- Упражнения (Поясница)
INSERT INTO "exercises" ("title", "categoryId", "statusId")
VALUES ('Супермен', 14, 1),
       ('Гиперэкстензия', 14, 1),
       ('Становая тяга', 14, 1);
