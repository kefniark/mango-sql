-- queryMany: TableNames
SELECT DISTINCT table_name
FROM information_schema.columns
WHERE table_schema = 'public';

-- queryMany: TableColumns
SELECT column_name, column_default, is_nullable, data_type, udt_name
FROM information_schema.columns
WHERE table_schema = 'public';
