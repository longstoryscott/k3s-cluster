#!/bin/bash
set -e

echo "Starting PostgreSQL database migration..."
DB_NAME="nextcloud"
DUMP_FILE="db.out"

echo "1. Creating dump from original PostgreSQL instance (postgres-0)..."
# kubectl exec -it postgres-0 -n psql -- pg_dump -U lsm -d ${DB_NAME} -c -C -b >${DUMP_FILE}
kubectl exec -it postgres-0 -n psql -- pg_dumpall -U lsm -c >${DUMP_FILE}

echo "✅ Database dump created at ${DUMP_FILE}"
echo "2. Copying dump file to new PostgreSQL pod (psql-0)..."
# Create a temporary directory in the pod
kubectl exec -it psql-0 -n psql -- mkdir -p /tmp/db_migration

# Copy the dump file to the pod
kubectl cp ${DUMP_FILE} psql/psql-0:/tmp/db_migration/${DUMP_FILE} -n psql
echo "✅ Dump file copied to new PostgreSQL pod"

echo "3. Creating required roles in the new PostgreSQL instance..."
# Create the oc_lsm role that's referenced in the dump
# kubectl exec -it psql-0 -n psql -- bash -c "PGPASSWORD=\$POSTGRES_PASSWORD psql -U lsm -d postgres -c \"CREATE ROLE oc_lsm WITH LOGIN PASSWORD '\$POSTGRES_PASSWORD';\""
echo "✅ Created necessary database roles"

echo "4. Restoring database in the new PostgreSQL instance..."
# Use psql to restore the plain SQL dump
# kubectl exec -it psql-0 -n psql -- bash -c "PGPASSWORD=\$POSTGRES_PASSWORD psql -U lsm -d postgres -c \"DROP DATABASE IF EXISTS ${DB_NAME};\""
# kubectl exec -it psql-0 -n psql -- bash -c "PGPASSWORD=\$POSTGRES_PASSWORD psql -U lsm -d postgres -c \"CREATE DATABASE ${DB_NAME};\""
# kubectl exec -it psql-0 -n psql -- bash -c "PGPASSWORD=\$POSTGRES_PASSWORD psql -U lsm -d nextcloud < /tmp/db_migration/${DB_NAME}.dump"
kubectl exec -it psql-0 -n psql -- bash -c "PGPASSWORD=\$POSTGRES_PASSWORD psql -U lsm -X -f /tmp/db_migration/${DUMP_FILE} -d postgres"
echo "✅ Database ${DB_NAME} restored successfully"

echo "5. Cleaning up temporary files..."
rm -f ${DUMP_FILE}
kubectl exec -it psql-0 -n psql -- rm -rf /tmp/db_migration
echo "✅ Temporary files cleaned up"

echo "Migration completed successfully! You can verify the data in the new instance using:"
echo "kubectl exec -it psql-0 -n psql -- psql -U lsm -d ${DB_NAME} -c \"SELECT COUNT(*) FROM information_schema.tables;\""
