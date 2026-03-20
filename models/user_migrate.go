package models

import (
	"encoding/json"
	"os"
)

// MigrateJSONToSQLite migrates users from a JSON file to SQLite database
func MigrateJSONToSQLite(jsonPath, sqlitePath string) error {
	// Read JSON file
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	// Parse JSON
	var users []User
	if err := json.Unmarshal(data, &users); err != nil {
		return err
	}

	// Create SQLite database
	sqliteDB, err := NewSQLiteUserDatabase(sqlitePath)
	if err != nil {
		return err
	}
	defer sqliteDB.Close()

	// Insert users
	for _, user := range users {
		// Check if user already exists
		if _, exists := sqliteDB.GetUser(user.Username); exists {
			continue // Skip existing users
		}

		// Create user (password is already hashed)
		// We need to insert directly without hashing again
		if err := sqliteDB.insertUserWithoutHash(&user); err != nil {
			return err
		}
	}

	return nil
}

// insertUserWithoutHash inserts a user without hashing the password (for migration)
func (ud *SQLiteUserDatabase) insertUserWithoutHash(user *User) error {
	ud.mu.Lock()
	defer ud.mu.Unlock()

	query := `INSERT OR REPLACE INTO users (id, username, password, email, role, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := ud.db.Exec(query,
		user.ID,
		user.Username,
		user.Password, // Already hashed
		user.Email,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)

	return err
}
