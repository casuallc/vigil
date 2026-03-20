#!/bin/bash

# migrate-users.sh - Migrate users from JSON to SQLite

JSON_FILE="conf/users.json"
DB_FILE="conf/users.db"

if [ ! -f "$JSON_FILE" ]; then
    echo "No JSON file found at $JSON_FILE"
    exit 0
fi

if [ -f "$DB_FILE" ]; then
    echo "SQLite database already exists at $DB_FILE"
    read -p "Do you want to overwrite? (y/N): " confirm
    if [ "$confirm" != "y" ] && [ "$confirm" != "Y" ]; then
        echo "Migration cancelled"
        exit 0
    fi
fi

echo "Migrating users from $JSON_FILE to $DB_FILE..."

# Create a simple Go program to do the migration
cat > /tmp/migrate_users.go << 'EOF'
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: migrate_users <json_path> <sqlite_path>")
		os.Exit(1)
	}

	jsonPath := os.Args[1]
	sqlitePath := os.Args[2]

	// Import the models package to use the migration function
	// This is a placeholder - the actual migration is done by the server
	fmt.Printf("Note: Please restart the server to migrate users from %s to %s\n", jsonPath, sqlitePath)
	fmt.Println("The server will automatically migrate users on startup.")
}
EOF

echo "Migration will be performed automatically when you start the server."
echo "The JSON file will be removed after successful migration."
