# Database Migration Guide

This guide provides instructions for migrating between different database backends in Ops-Butler.

## Supported Databases

Ops-Butler supports the following database backends:

1. **SQLite** (default) - Embedded database, ideal for simple deployments
2. **PostgreSQL** - For larger deployments or when external database is preferred
3. **MySQL** - Alternative external database option

## Migration Scenarios

### 1. SQLite to External Database (PostgreSQL/MySQL)

This is the most common migration path when scaling up from a simple deployment.

#### Prerequisites

- Running Ops-Butler instance with SQLite
- PostgreSQL or MySQL server accessible from the Kubernetes cluster
- Database credentials with permissions to create tables

#### Step 1: Create the Target Database

For PostgreSQL:
```sql
CREATE DATABASE ops_butler;
CREATE USER ops_butler WITH PASSWORD 'your-password';
GRANT ALL PRIVILEGES ON DATABASE ops_butler TO ops_butler;
```

For MySQL:
```sql
CREATE DATABASE ops_butler;
CREATE USER 'ops_butler'@'%' IDENTIFIED BY 'your-password';
GRANT ALL PRIVILEGES ON ops_butler.* TO 'ops_butler'@'%';
FLUSH PRIVILEGES;
```

#### Step 2: Export Data from SQLite

1. Create a backup of your SQLite database:

```bash
# Get a shell to the Core pod
kubectl exec -it -n ops-butler ops-butler-core-0 -- /bin/sh

# Inside the pod, backup the database
sqlite3 /data/ops-butler.db .dump > /data/ops-butler-backup.sql
```

2. Copy the backup file to your local machine:

```bash
kubectl cp ops-butler/ops-butler-core-0:/data/ops-butler-backup.sql ./ops-butler-backup.sql
```

#### Step 3: Convert SQLite Schema to Target Database

The schema needs to be adjusted for the target database. You can use tools like `pgloader` for PostgreSQL or manually modify the schema.

For PostgreSQL using pgloader:
```bash
pgloader sqlite:///path/to/ops-butler-backup.sql postgresql://ops_butler:your-password@your-postgres-host/ops_butler
```

For manual conversion, you'll need to:
1. Replace SQLite-specific types with PostgreSQL/MySQL types
2. Adjust auto-increment syntax
3. Modify index definitions

#### Step 4: Update Ops-Butler Configuration

1. Edit the ConfigMap in `deploy/k8s/core.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ops-butler-core-config
  namespace: ops-butler
data:
  # For PostgreSQL
  DB_TYPE: "postgres"
  DB_HOST: "your-postgres-host"
  DB_PORT: "5432"
  DB_USER: "ops_butler"
  DB_PASSWORD: "your-password"
  DB_NAME: "ops_butler"
  DB_SSL_MODE: "disable"
  
  # For MySQL
  # DB_TYPE: "mysql"
  # DB_HOST: "your-mysql-host"
  # DB_PORT: "3306"
  # DB_USER: "ops_butler"
  # DB_PASSWORD: "your-password"
  # DB_NAME: "ops_butler"
```

2. Apply the changes:

```bash
kubectl apply -f deploy/k8s/core.yaml
```

3. Restart the Core pod:

```bash
kubectl rollout restart statefulset -n ops-butler ops-butler-core
```

#### Step 5: Verify Migration

1. Check the logs to ensure the Core pod connected to the external database:

```bash
kubectl logs -n ops-butler ops-butler-core-0
```

2. Use the Web UI to verify that all data is accessible

### 2. External Database to SQLite

This migration is less common but might be needed for simplifying deployments.

#### Prerequisites

- Running Ops-Butler instance with PostgreSQL or MySQL
- Sufficient disk space for SQLite database

#### Step 1: Export Data from External Database

For PostgreSQL:
```bash
pg_dump -h your-postgres-host -U ops_butler -d ops_butler -f ops-butler-backup.sql
```

For MySQL:
```bash
mysqldump -h your-mysql-host -u ops_butler -p ops_butler > ops-butler-backup.sql
```

#### Step 2: Convert Schema to SQLite

You'll need to manually adjust the schema for SQLite compatibility:
1. Replace PostgreSQL/MySQL-specific types with SQLite types
2. Adjust auto-increment syntax
3. Modify index definitions

#### Step 3: Import Data to SQLite

1. Create a new SQLite database:

```bash
sqlite3 ops-butler.db < converted-schema.sql
```

2. Copy the database to the Core pod:

```bash
kubectl cp ./ops-butler.db ops-butler/ops-butler-core-0:/data/ops-butler.db
```

#### Step 4: Update Ops-Butler Configuration

1. Edit the ConfigMap in `deploy/k8s/core.yaml`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ops-butler-core-config
  namespace: ops-butler
data:
  DB_TYPE: "sqlite"
  DB_PATH: "/data/ops-butler.db"
  # Remove other DB_* environment variables
```

2. Apply the changes:

```bash
kubectl apply -f deploy/k8s/core.yaml
```

3. Restart the Core pod:

```bash
kubectl rollout restart statefulset -n ops-butler ops-butler-core
```

## Backup and Restore

### SQLite Backup

1. Create a backup of your SQLite database:

```bash
kubectl exec -it -n ops-butler ops-butler-core-0 -- /bin/sh
sqlite3 /data/ops-butler.db .dump > /data/ops-butler-backup.sql
```

2. Copy the backup file to your local machine:

```bash
kubectl cp ops-butler/ops-butler-core-0:/data/ops-butler-backup.sql ./ops-butler-backup.sql
```

### PostgreSQL Backup

```bash
pg_dump -h your-postgres-host -U ops_butler -d ops_butler -f ops-butler-backup.sql
```

### MySQL Backup

```bash
mysqldump -h your-mysql-host -u ops_butler -p ops_butler > ops-butler-backup.sql
```

### Restore from Backup

#### SQLite Restore

```bash
kubectl cp ./ops-butler-backup.sql ops-butler/ops-butler-core-0:/data/
kubectl exec -it -n ops-butler ops-butler-core-0 -- /bin/sh
cat /data/ops-butler-backup.sql | sqlite3 /data/ops-butler.db
```

#### PostgreSQL Restore

```bash
psql -h your-postgres-host -U ops_butler -d ops_butler -f ops-butler-backup.sql
```

#### MySQL Restore

```bash
mysql -h your-mysql-host -u ops_butler -p ops_butler < ops-butler-backup.sql
```

## Troubleshooting

### Migration Fails Due to Schema Differences

If migration fails due to schema differences:

1. Check the logs for specific errors:
```bash
kubectl logs -n ops-butler ops-butler-core-0
```

2. Manually adjust the schema in your backup file
3. Retry the migration

### Data Type Conversion Issues

Different databases handle data types differently. Common issues include:

1. **Boolean values**: SQLite doesn't have a native boolean type
2. **JSON data**: Handling of JSON differs between databases
3. **Date/time formats**: Format and timezone handling varies

For these issues, you may need to transform the data during migration.

### Connection Issues

If Ops-Butler can't connect to the external database:

1. Verify the database credentials
2. Check network connectivity from the Kubernetes cluster to the database
3. Ensure firewall rules allow the connection
4. Verify the database server is configured to accept remote connections

## Best Practices

1. **Always backup** your data before migration
2. **Test migrations** in a non-production environment first
3. **Schedule migrations** during low-traffic periods
4. **Monitor the application** closely after migration
5. **Keep a backup** of the old database until you're confident the migration was successful