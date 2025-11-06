package database

import (
    "context"
    "fmt"
    "log"

    "github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
    pool *pgxpool.Pool
}

func NewDatabase(ctx context.Context, connString string) (*Database, error) {
    config, err := pgxpool.ParseConfig(connString)
    if err != nil {
        return nil, fmt.Errorf("не удалось распарсить строку подключения: %w", err)
    }

    config.MaxConns = 10
    config.MinConns = 2

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("не удалось создать пул соединений: %w", err)
    }

    if err := pool.Ping(ctx); err != nil {
        return nil, fmt.Errorf("не удалось подключиться к базе данных: %w", err)
    }

    log.Println("Успешное подключение к PostgreSQL")

    return &Database{pool: pool}, nil
}

func (db *Database) Close() {
    db.pool.Close()
}

func (db *Database) GetPool() *pgxpool.Pool {
    return db.pool
}
