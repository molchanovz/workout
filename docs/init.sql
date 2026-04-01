INSERT INTO "statuses" ( "statusId", "title", "alias" ) VALUES ( 1, 'Опубликован', 'enabled' );
INSERT INTO "statuses" ( "statusId", "title", "alias" ) VALUES ( 2, 'Не опубликован', 'disabled' );
INSERT INTO "statuses" ( "statusId", "title", "alias" ) VALUES ( 3, 'Удален', 'deleted' );

-- password is 12345
INSERT INTO "users" ( "login", "password", "statusId" ) VALUES ( 'admin', '$2y$14$4IpqlaJ2Rvfgs.wb8f6lPODVLb/Ygl6zw1ZCUKz5CuT6WB6CV44AG', 1 );

INSERT INTO "vfsFolders" ("parentFolderId", title, "isFavorite", "createdAt", "statusId") VALUES (null, 'root', false, now(), 1);

-- Корневые категории
INSERT INTO "categories" ("title", "statusId") VALUES ('Верх тела', 1), ('Низ тела', 1);

-- Подкатегории верха (parentCategoryId = 1)
INSERT INTO "categories" ("title", "parentCategoryId", "statusId") VALUES
  ('Грудь', 1, 1), ('Спина', 1, 1), ('Руки', 1, 1);

-- Подкатегория низа (parentCategoryId = 2)
INSERT INTO "categories" ("title", "parentCategoryId", "statusId") VALUES ('Ноги', 2, 1);

-- Базовые упражнения
INSERT INTO "exercises" ("title", "categoryId", "statusId") VALUES
  ('Жим лёжа', 3, 1), ('Жим гантелей', 3, 1),
  ('Подтягивания', 4, 1), ('Тяга блока', 4, 1),
  ('Сгибание на бицепс', 5, 1), ('Разгибание трицепс', 5, 1),
  ('Приседания', 6, 1), ('Жим ногами', 6, 1);
