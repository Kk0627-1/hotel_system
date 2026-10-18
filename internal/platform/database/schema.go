package database

import (
	"database/sql"
	"fmt"
	"log"
)

// InitSchema creates application tables when they do not already exist.
func InitSchema(db *sql.DB) error {
	if err := initUserTable(db); err != nil {
		return err
	}
	if err := initWalletTable(db); err != nil {
		return err
	}
	if err := initHotelTable(db); err != nil {
		return err
	}
	if err := initRoomTable(db); err != nil {
		return err
	}
	return initRecordTable(db)
}

func ensureForeignKey(db *sql.DB, table, constraint, definition string) error {
	query := fmt.Sprintf(`
DO $$ BEGIN
  ALTER TABLE %s ADD CONSTRAINT %s %s;
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;
`, table, constraint, definition)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("add foreign key %s failed: %v", constraint, err)
		return err
	}
	return nil
}

func initUserTable(db *sql.DB) error {
	query := `
	  CREATE TABLE IF NOT EXISTS users (
	    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(45) NOT NULL UNIQUE,
		password VARCHAR(45) NOT NULL,
		regist_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	  );
	  CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`
	if _, err := db.Exec(query); err != nil {
		log.Printf("create users table failed: %v", err)
		return err
	}
	log.Println("users table ready")
	return nil
}

func initWalletTable(db *sql.DB) error {
	query := `
	  CREATE TABLE IF NOT EXISTS wallets (
	    user_id UUID PRIMARY KEY,
		balance DECIMAL(12, 2) NOT NULL DEFAULT 0,
		create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		update_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_wallets_user FOREIGN KEY (user_id) REFERENCES users(id),
		CONSTRAINT chk_wallets_balance_non_negative CHECK (balance >= 0)
	  );
	`
	if _, err := db.Exec(query); err != nil {
		log.Printf("create wallets table failed: %v", err)
		return err
	}
	if err := ensureForeignKey(db, "wallets", "fk_wallets_user", "FOREIGN KEY (user_id) REFERENCES users(id)"); err != nil {
		return err
	}
	log.Println("wallets table ready")
	return nil
}

func initHotelTable(db *sql.DB) error {
	query := `
	  CREATE TABLE IF NOT EXISTS hotels (
	    id SERIAL PRIMARY KEY,
		name VARCHAR(45) NOT NULL UNIQUE,
        create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		status BOOLEAN NOT NULL DEFAULT FALSE
	  );
	  CREATE INDEX IF NOT EXISTS idx_hotels_id ON hotels(id);
	`
	if _, err := db.Exec(query); err != nil {
		log.Printf("create hotels table failed: %v", err)
		return err
	}
	log.Println("hotels table ready")
	return nil
}

func initRoomTable(db *sql.DB) error {
	query := `
	  CREATE TABLE IF NOT EXISTS rooms (
	    id SERIAL PRIMARY KEY,
		hotel_id INT NOT NULL,
		room_type VARCHAR(45) NOT NULL,
		price DECIMAL(10, 2) NOT NULL,
		quantity INT NOT NULL DEFAULT 1,
		status BOOLEAN NOT NULL DEFAULT TRUE,
		create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(hotel_id, room_type),
		CONSTRAINT fk_rooms_hotel FOREIGN KEY (hotel_id) REFERENCES hotels(id)
	  );
	  CREATE INDEX IF NOT EXISTS idx_rooms_id ON rooms(id);
	  CREATE INDEX IF NOT EXISTS idx_rooms_hotel_id ON rooms(hotel_id);
	`
	if _, err := db.Exec(query); err != nil {
		log.Printf("create rooms table failed: %v", err)
		return err
	}
	if err := ensureForeignKey(db, "rooms", "fk_rooms_hotel", "FOREIGN KEY (hotel_id) REFERENCES hotels(id)"); err != nil {
		return err
	}
	log.Println("rooms table ready")
	return nil
}

func initRecordTable(db *sql.DB) error {
	query := `
	  CREATE TABLE IF NOT EXISTS records (
	    id SERIAL PRIMARY KEY,
		user_id UUID NOT NULL,
		hotel_id INT NOT NULL,
		room_id INT NOT NULL,
		booking_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checkin_time TIMESTAMP NOT NULL,
		checkout_time TIMESTAMP NOT NULL,
		total_price DECIMAL(10, 2) NOT NULL,
		booking_num INT NOT NULL DEFAULT 1,
		payment_type VARCHAR(45) NOT NULL DEFAULT 'cash',
		payment_status VARCHAR(20) NOT NULL DEFAULT 'pending',
		status BOOLEAN NOT NULL DEFAULT TRUE,
		create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_records_user FOREIGN KEY (user_id) REFERENCES users(id),
		CONSTRAINT fk_records_hotel FOREIGN KEY (hotel_id) REFERENCES hotels(id),
		CONSTRAINT fk_records_room FOREIGN KEY (room_id) REFERENCES rooms(id)
	  );
	  CREATE INDEX IF NOT EXISTS idx_records_id ON records(id);
	  CREATE INDEX IF NOT EXISTS idx_records_payment_status ON records(payment_status);
	  CREATE INDEX IF NOT EXISTS idx_records_user_payment ON records(user_id, payment_status);
	`
	if _, err := db.Exec(query); err != nil {
		log.Printf("create records table failed: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ADD COLUMN IF NOT EXISTS checkin_time TIMESTAMP`); err != nil {
		log.Printf("add checkin_time failed: %v", err)
		return err
	}
	if _, err := db.Exec(`UPDATE records SET checkin_time = booking_time WHERE checkin_time IS NULL`); err != nil {
		log.Printf("backfill checkin_time failed: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ALTER COLUMN checkin_time SET NOT NULL`); err != nil {
		log.Printf("set checkin_time not null failed: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ADD COLUMN IF NOT EXISTS booking_num INT`); err != nil {
		log.Printf("add booking_num failed: %v", err)
		return err
	}
	if _, err := db.Exec(`UPDATE records SET booking_num = 1 WHERE booking_num IS NULL`); err != nil {
		log.Printf("backfill booking_num failed: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ALTER COLUMN booking_num SET DEFAULT 1`); err != nil {
		log.Printf("set booking_num default failed: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ALTER COLUMN booking_num SET NOT NULL`); err != nil {
		log.Printf("set booking_num not null failed: %v", err)
		return err
	}
	if err := ensureForeignKey(db, "records", "fk_records_user", "FOREIGN KEY (user_id) REFERENCES users(id)"); err != nil {
		return err
	}
	if err := ensureForeignKey(db, "records", "fk_records_hotel", "FOREIGN KEY (hotel_id) REFERENCES hotels(id)"); err != nil {
		return err
	}
	if err := ensureForeignKey(db, "records", "fk_records_room", "FOREIGN KEY (room_id) REFERENCES rooms(id)"); err != nil {
		return err
	}
	log.Println("records table ready")
	return nil
}
