package privatekey

import "github.com/jackc/pgx/v5/pgxpool"

type PrivateKeyHandler struct {
	DB *pgxpool.Pool
}
