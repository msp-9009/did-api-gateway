#!/bin/bash
# Automated PostgreSQL Backup Script
# Phase 4: Reliability & Resilience

set -eo pipefail

# Configuration
BACKUP_DIR="${BACKUP_DIR:-/backups/postgres}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"
S3_BUCKET="${S3_BUCKET:-}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-gateway}"
POSTGRES_DB="${POSTGRES_DB:-gateway}"

# Generate timestamp
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="postgres_backup_${TIMESTAMP}.sql.gz"
BACKUP_PATH="$BACKUP_DIR/$BACKUP_FILE"

echo "============================================"
echo "PostgreSQL Backup Started"
echo "============================================"
echo "Timestamp: $TIMESTAMP"
echo "Database: $POSTGRES_DB@$POSTGRES_HOST:$POSTGRES_PORT"
echo "Backup file: $BACKUP_PATH"
echo ""

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Run pg_dump with compression
echo "Creating backup..."
PGPASSWORD="$POSTGRES_PASSWORD" pg_dump \
    -h "$POSTGRES_HOST" \
    -p "$POSTGRES_PORT" \
    -U "$POSTGRES_USER" \
    -d "$POSTGRES_DB" \
    --format=custom \
    --compress=9 \
    --verbose \
    2>&1 | gzip > "$BACKUP_PATH"

# Check if backup was successful
if [ $? -eq 0 ] && [ -f "$BACKUP_PATH" ]; then
    BACKUP_SIZE=$(du -h "$BACKUP_PATH" | cut -f1)
    echo ""
    echo "✅ Backup created successfully: $BACKUP_SIZE"
else
    echo ""
    echo "❌ Backup failed!"
    exit 1
fi

# Upload to S3 if configured
if [ -n "$S3_BUCKET" ]; then
    echo ""
    echo "Uploading to S3: s3://$S3_BUCKET/postgres-backups/"
    
    aws s3 cp "$BACKUP_PATH" "s3://$S3_BUCKET/postgres-backups/" \
        --storage-class STANDARD_IA \
        --metadata "timestamp=$TIMESTAMP,database=$POSTGRES_DB"
    
    if [ $? -eq 0 ]; then
        echo "✅ Backup uploaded to S3"
        
        # Create a 'latest' symlink in S3
        aws s3 cp "s3://$S3_BUCKET/postgres-backups/$BACKUP_FILE" \
                  "s3://$S3_BUCKET/postgres-backups/latest.sql.gz"
    else
        echo "⚠️  S3 upload failed (backup still saved locally)"
    fi
fi

# Clean up old backups
echo ""
echo "Cleaning up backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -name "postgres_backup_*.sql.gz" -mtime +$RETENTION_DAYS -delete
REMAINING=$(find "$BACKUP_DIR" -name "postgres_backup_*.sql.gz" | wc -l)
echo "Remaining backups: $REMAINING"

# Also clean up old S3 backups if configured
if [ -n "$S3_BUCKET" ]; then
    CUTOFF_DATE=$(date -d "$RETENTION_DAYS days ago" +%Y%m%d)
    echo "S3 cleanup: Removing backups older than $CUTOFF_DATE"
    
    aws s3 ls "s3://$S3_BUCKET/postgres-backups/" | while read -r line; do
        FILE_DATE=$(echo "$line" | awk '{print $4}' | grep -oP '\d{8}' | head -1)
        FILE_NAME=$(echo "$line" | awk '{print $4}')
        
        if [[ "$FILE_DATE" < "$CUTOFF_DATE" ]] && [[ "$FILE_NAME" != "latest.sql.gz" ]]; then
            echo "Deleting old backup: $FILE_NAME"
            aws s3 rm "s3://$S3_BUCKET/postgres-backups/$FILE_NAME"
        fi
    done
fi

echo ""
echo "============================================"
echo "Backup Complete!"
echo "============================================"
echo "Local path: $BACKUP_PATH"
[ -n "$S3_BUCKET" ] && echo "S3 path: s3://$S3_BUCKET/postgres-backups/$BACKUP_FILE"
echo "Retention: $RETENTION_DAYS days"
echo ""
