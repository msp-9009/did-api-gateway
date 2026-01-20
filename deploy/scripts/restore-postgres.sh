#!/bin/bash
# PostgreSQL Restore Script
# Phase 4: Reliability & Resilience

set -eo pipefail

# Configuration
BACKUP_FILE="${1:-}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-gateway}"
POSTGRES_DB="${POSTGRES_DB:-gateway}"
S3_BUCKET="${S3_BUCKET:-}"

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup-file>"
    echo ""
    echo "Examples:"
    echo "  $0 /backups/postgres/postgres_backup_20260120_120000.sql.gz"
    echo "  $0 s3://my-bucket/postgres-backups/latest.sql.gz"
    echo ""
    exit 1
fi

echo "============================================"
echo "PostgreSQL Restore Started"
echo "============================================"
echo "Target database: $POSTGRES_DB@$POSTGRES_HOST:$POSTGRES_PORT"
echo "Source: $BACKUP_FILE"
echo ""

# Download from S3 if needed
if [[ "$BACKUP_FILE" == s3://* ]]; then
    echo "Downloading from S3..."
    TEMP_FILE="/tmp/postgres_restore_$(date +%s).sql.gz"
    aws s3 cp "$BACKUP_FILE" "$TEMP_FILE"
    BACKUP_FILE="$TEMP_FILE"
fi

# Check if backup file exists
if [ ! -f "$BACKUP_FILE" ]; then
    echo "❌ Error: Backup file not found: $BACKUP_FILE"
    exit 1
fi

echo "Backup file size: $(du -h "$BACKUP_FILE" | cut -f1)"
echo ""

# Confirm before proceeding
read -p "⚠️  This will REPLACE all data in $POSTGRES_DB. Continue? (yes/no): " -r
echo
if [[ ! $REPLY =~ ^yes$ ]]; then
    echo "Restore cancelled."
    exit 0
fi

# Drop existing connections
echo "Terminating existing connections..."
PGPASSWORD="$POSTGRES_PASSWORD" psql \
    -h "$POSTGRES_HOST" \
    -p "$POSTGRES_PORT" \
    -U "$POSTGRES_USER" \
    -d postgres \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$POSTGRES_DB' AND pid <> pg_backend_pid();" \
    || true

# Drop and recreate database
echo "Recreating database..."
PGPASSWORD="$POSTGRES_PASSWORD" psql \
    -h "$POSTGRES_HOST" \
    -p "$POSTGRES_PORT" \
    -U "$POSTGRES_USER" \
    -d postgres \
    -c "DROP DATABASE IF EXISTS $POSTGRES_DB;"

PGPASSWORD="$POSTGRES_PASSWORD" psql \
    -h "$POSTGRES_HOST" \
    -p "$POSTGRES_PORT" \
    -U "$POSTGRES_USER" \
    -d postgres \
    -c "CREATE DATABASE $POSTGRES_DB;"

# Restore from backup
echo ""
echo "Restoring data..."
gunzip < "$BACKUP_FILE" | \
    PGPASSWORD="$POSTGRES_PASSWORD" pg_restore \
    -h "$POSTGRES_HOST" \
    -p "$POSTGRES_PORT" \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    --verbose \
    --no-owner \
    --no-acl

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Restore completed successfully!"
else
    echo ""
    echo "❌ Restore failed!"
    exit 1
fi

# Verify restored data
echo ""
echo "Verifying restored data..."
POLICY_COUNT=$(PGPASSWORD="$POSTGRES_PASSWORD" psql \
    -h "$POSTGRES_HOST" \
    -p "$POSTGRES_PORT" \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    -t -c "SELECT count(*) FROM policies;")

echo "Policies: $POLICY_COUNT"

# Clean up temp file if we downloaded from S3
if [[ -n "$TEMP_FILE" ]] && [[ -f "$TEMP_FILE" ]]; then
    rm "$TEMP_FILE"
fi

echo ""
echo "============================================"
echo "Restore Complete!"
echo "============================================"
echo ""
echo "Next steps:"
echo "1. Restart gateway pods: kubectl rollout restart deployment/did-gateway"
echo "2. Verify application functionality"
echo "3. Check logs: kubectl logs -f deployment/did-gateway"
echo ""
