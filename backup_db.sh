#!/bin/bash

# Database backup script for torrent_seeker
# Creates a timestamped backup of the PostgreSQL database

# Configuration
DB_NAME="torrent_seeker"
BACKUP_DIR="./backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILE="${BACKUP_DIR}/torrent_seeker_${TIMESTAMP}.sql"

# Create backup directory if it doesn't exist
mkdir -p "$BACKUP_DIR"

# Perform the backup
echo "Starting backup of database: $DB_NAME"
echo "Backup file: $BACKUP_FILE"

pg_dump "$DB_NAME" > "$BACKUP_FILE"

if [ $? -eq 0 ]; then
    echo "Backup completed successfully!"
    echo "File size: $(du -h "$BACKUP_FILE" | cut -f1)"
    
    # Optional: Compress the backup
    gzip "$BACKUP_FILE"
    echo "Backup compressed: ${BACKUP_FILE}.gz"
    
    # Optional: Keep only last 7 backups
    echo "Cleaning up old backups (keeping last 7)..."
    ls -t "${BACKUP_DIR}"/torrent_seeker_*.sql.gz | tail -n +8 | xargs -r rm
    
    echo "Done!"
else
    echo "Backup failed!"
    exit 1
fi
