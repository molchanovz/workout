CREATE
OR REPLACE FUNCTION lower_ru(text) RETURNS text AS $$
SELECT translate($1,
                 'АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯ',
                 'абвгдеёжзийклмнопрстуфхцчшщъыьэюя');
$$
LANGUAGE SQL IMMUTABLE;