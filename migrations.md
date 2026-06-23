# Migration Guide

This project uses SQL migration files in the `migrations/` folder.

## 1. Install Migration CLI

Install `golang-migrate`:

```bash
brew install golang-migrate
```

Check that it is installed:

```bash
migrate -version
```

## 2. Check Database Config

Make sure your `.env` has the correct database values:

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=password
DB_NAME=cermin_db
DB_SSLMODE=disable
```

The `Makefile` uses these values to build the database URL.

## 3. Create A New Migration

Example for a products table:

```bash
make migrate-create name=create_products_table
```

This creates two files:

```txt
migrations/000002_create_products_table.up.sql
migrations/000002_create_products_table.down.sql
```

## 4. Write The Up Migration

The `.up.sql` file applies the database change.

Example:

```sql
CREATE TABLE products (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    price BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## 5. Write The Down Migration

The `.down.sql` file rolls back the database change.

Example:

```sql
DROP TABLE IF EXISTS products;
```

## 6. Run Migrations

Apply all pending migrations:

```bash
make migrate-up
```

## 7. Check Migration Version

```bash
make migrate-version
```

## 8. Roll Back One Migration

```bash
make migrate-down
```

## 9. Fix Dirty Migration State

If a migration fails halfway, the database can become dirty.

Force the database to a specific version:

```bash
make migrate-force version=1
```

Then run migrations again:

```bash
make migrate-up
```

## Daily Workflow

When adding a new model or table:

```txt
1. Create or update the Go model
2. Create a migration file
3. Write SQL in the .up.sql file
4. Write rollback SQL in the .down.sql file
5. Run the migration
6. Write repository queries
7. Test the feature
```

Example:

```bash
make migrate-create name=create_orders_table
make migrate-up
make migrate-version
```

## Important Rule

Changing a Go model does not automatically change the database.

You must create and run a migration for database structure changes.
