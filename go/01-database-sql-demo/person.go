package main

import (
	"context"
	"database/sql"
	"fmt"
)

func insertPerson(ctx context.Context, db *sql.DB, person Person) (int64, error) {
	result, err := db.ExecContext(ctx,
		"INSERT INTO person (username, age, email) VALUES (?, ?, ?)",
		person.Username, person.Age, person.Email)
	if err != nil {
		return 0, fmt.Errorf("insert person: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted id: %w", err)
	}
	return id, nil
}

func findPerson(ctx context.Context, db *sql.DB, id int64) (Person, error) {
	var person Person
	err := db.QueryRowContext(ctx,
		"SELECT user_id, username, age, email FROM person WHERE user_id = ?", id,
	).Scan(&person.UserID, &person.Username, &person.Age, &person.Email)
	if err != nil {
		return Person{}, fmt.Errorf("find person %d: %w", id, err)
	}
	return person, nil
}

func listPeople(ctx context.Context, db *sql.DB) ([]Person, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT user_id, username, age, email FROM person ORDER BY user_id")
	if err != nil {
		return nil, fmt.Errorf("query people: %w", err)
	}
	defer rows.Close()
	people := make([]Person, 0)
	for rows.Next() {
		var person Person
		if err := rows.Scan(&person.UserID, &person.Username, &person.Age, &person.Email); err != nil {
			return nil, fmt.Errorf("scan person: %w", err)
		}
		people = append(people, person)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate people: %w", err)
	}
	return people, nil
}

func updateAge(ctx context.Context, db *sql.DB, id int64, age int) (int64, error) {
	result, err := db.ExecContext(ctx,
		"UPDATE person SET age = ? WHERE user_id = ?", age, id)
	if err != nil {
		return 0, fmt.Errorf("update person: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read update result: %w", err)
	}
	return affected, nil
}

func deletePerson(ctx context.Context, db *sql.DB, id int64) (int64, error) {
	result, err := db.ExecContext(ctx, "DELETE FROM person WHERE user_id = ?", id)
	if err != nil {
		return 0, fmt.Errorf("delete person: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("read delete result: %w", err)
	}
	return affected, nil
}

func createPeople(ctx context.Context, db *sql.DB, people []Person) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	for _, person := range people {
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO person (username, age, email) VALUES (?, ?, ?)",
			person.Username, person.Age, person.Email); err != nil {
			return fmt.Errorf("insert %s: %w", person.Username, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func findByUsername(ctx context.Context, db *sql.DB, username string) (Person, error) {
	var person Person
	err := db.QueryRowContext(ctx, "SELECT user_id, username, age, email FROM person WHERE username = ?", username).Scan(&person.UserID, &person.Username, &person.Age, &person.Email)
	if err != nil {
		return Person{}, fmt.Errorf("find username %q: %w", username, err)
	}
	return person, nil
}
