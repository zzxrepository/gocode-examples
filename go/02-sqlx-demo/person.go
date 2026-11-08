package main

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

func insertPerson(ctx context.Context, db *sqlx.DB, person Person) (int64, error) {
	result, err := db.NamedExecContext(ctx,
		"INSERT INTO person (username, age, email) VALUES (:username, :age, :email)", person)
	if err != nil {
		return 0, fmt.Errorf("insert person: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read inserted id: %w", err)
	}
	return id, nil
}

func findPerson(ctx context.Context, db *sqlx.DB, id int64) (Person, error) {
	var person Person
	err := db.GetContext(ctx, &person,
		"SELECT user_id, username, age, email FROM person WHERE user_id = ?", id)
	if err != nil {
		return Person{}, fmt.Errorf("find person %d: %w", id, err)
	}
	return person, nil
}

func listPeople(ctx context.Context, db *sqlx.DB) ([]Person, error) {
	people := make([]Person, 0)
	err := db.SelectContext(ctx, &people,
		"SELECT user_id, username, age, email FROM person ORDER BY user_id")
	if err != nil {
		return nil, fmt.Errorf("list people: %w", err)
	}
	return people, nil
}

func updateAge(ctx context.Context, db *sqlx.DB, id int64, age int) (int64, error) {
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

func deletePerson(ctx context.Context, db *sqlx.DB, id int64) (int64, error) {
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

func createPeople(ctx context.Context, db *sqlx.DB, people []Person) error {
	tx, err := db.BeginTxx(ctx, nil)
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

func findPeopleByIDs(ctx context.Context, db *sqlx.DB, ids []int64) ([]Person, error) {
	if len(ids) == 0 {
		return []Person{}, nil
	}
	query, args, err := sqlx.In(
		"SELECT user_id, username, age, email FROM person WHERE user_id IN (?) ORDER BY user_id", ids)
	if err != nil {
		return nil, fmt.Errorf("expand ids: %w", err)
	}
	query = db.Rebind(query)
	people := make([]Person, 0)
	if err := db.SelectContext(ctx, &people, query, args...); err != nil {
		return nil, fmt.Errorf("query people by ids: %w", err)
	}
	return people, nil
}

func findByUsername(ctx context.Context, db *sqlx.DB, username string) (Person, error) {
	var person Person
	err := db.GetContext(ctx, &person, "SELECT user_id, username, age, email FROM person WHERE username = ?", username)
	if err != nil {
		return Person{}, fmt.Errorf("find username %q: %w", username, err)
	}
	return person, nil
}

// walkPeople 逐行调用 visit，不把整个结果集收集到切片。
func walkPeople(ctx context.Context, db *sqlx.DB, visit func(Person) error) error {
	rows, err := db.QueryxContext(ctx, "SELECT user_id, username, age, email FROM person ORDER BY user_id")
	if err != nil {
		return fmt.Errorf("query people: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var person Person
		if err := rows.StructScan(&person); err != nil {
			return fmt.Errorf("scan person: %w", err)
		}
		if err := visit(person); err != nil {
			return err
		}
	}
	return rows.Err()
}
