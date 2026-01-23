# Database Migrations

This directory contains SQL migration scripts for the NSW workflow management system.

## Migration Files

- `001_initial_schema.sql` - Creates all required tables, indexes, and constraints
- `001_initial_schema_down.sql` - Rollback script for the initial schema

## Running Migrations

### Option 1: Using psql (Manual)

```bash
# Set environment variables
export DB_HOST=localhost
export DB_PORT=5432
export DB_USERNAME=postgres
export DB_PASSWORD=password
export DB_NAME=nsw_db

# Run the migration
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USERNAME -d $DB_NAME -f internal/database/migrations/001_initial_schema.sql
```

### Option 2: Using the Application (Automatic)

The application will automatically run migrations on startup when using the database package.

### Rollback

To rollback the migration:

```bash
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USERNAME -d $DB_NAME -f internal/database/migrations/001_initial_schema_down.sql
```

## Database Schema

### Tables

1. **hs_codes** - Harmonized System codes for product classification
   - Primary key: `id` (UUID)
   - Unique constraint on `code`

2. **workflow_templates** - Workflow template definitions
   - Primary key: `id` (UUID)
   - Contains JSONB `steps` array

3. **workflow_template_maps** - Maps HS codes to workflow templates
   - Primary key: `id` (UUID)
   - Foreign keys: `hs_code_id`, `workflow_template_id`
   - Unique constraint on `(hs_code_id, type)`

4. **consignments** - Consignment records
   - Primary key: `id` (UUID)
   - Contains JSONB `items` array
   - States: `IN_PROGRESS`, `REQUIRES_REWORK`, `FINISHED`

5. **tasks** - Workflow task instances
   - Primary key: `id` (UUID)
   - Foreign key: `consignment_id`
   - Contains JSONB `config` and `depends_on` fields
   - Statuses: `LOCKED`, `READY`, `IN_PROGRESS`, `COMPLETED`, `REJECTED`

## Notes

- All tables use UUID primary keys with automatic generation
- Timestamps (`created_at`, `updated_at`) are automatically managed
- JSONB columns have GIN indexes for efficient querying
- Foreign key constraints ensure referential integrity with CASCADE delete
- Check constraints enforce valid enum values
