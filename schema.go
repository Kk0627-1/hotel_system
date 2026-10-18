package main

import (
	"fmt"
	"log"
)

// ensureForeignKey 為既有表補上外鍵（已存在則忽略）
func ensureForeignKey(table, constraint, definition string) error {
	query := fmt.Sprintf(`
DO $$ BEGIN
  ALTER TABLE %s ADD CONSTRAINT %s %s;
EXCEPTION
  WHEN duplicate_object THEN NULL;
END $$;
`, table, constraint, definition)
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("添加外键 %s 失败: %v", constraint, err)
		return err
	}
	return nil
}

// InitUserTable 初始化用戶表
func InitUserTable() error {
	query := `
	  CREATE TABLE IF NOT EXISTS users (
	    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		username VARCHAR(45) NOT NULL UNIQUE,
		password VARCHAR(45) NOT NULL,
		regist_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	  );
	  CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("创建用户表失败: %v", err)
		return err
	}
	log.Println("用户表创建成功")
	return nil
}

// InitWalletTable 初始化錢包表（每位使用者一筆，餘額預設 0）
func InitWalletTable() error {
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
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("创建錢包表失败: %v", err)
		return err
	}
	if err := ensureForeignKey("wallets", "fk_wallets_user", "FOREIGN KEY (user_id) REFERENCES users(id)"); err != nil {
		return err
	}
	log.Println("錢包表创建成功")
	return nil
}

// InitHotelTable 初始化飯店表
func InitHotelTable() error {
	query := `
	  CREATE TABLE IF NOT EXISTS hotels (
	    id SERIAL PRIMARY KEY,
		name VARCHAR(45) NOT NULL UNIQUE,
        create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		status BOOLEAN NOT NULL DEFAULT FALSE
	  );
	  CREATE INDEX IF NOT EXISTS idx_hotels_id ON hotels(id);
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("创建飯店表失败: %v", err)
		return err
	}
	log.Println("飯店表创建成功")
	return nil
}

// InitRoomTable 初始化房間表
func InitRoomTable() error {
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
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("创建房间表失败: %v", err)
		return err
	}
	// 既有表（當初無 FK）補上外鍵
	if err := ensureForeignKey("rooms", "fk_rooms_hotel", "FOREIGN KEY (hotel_id) REFERENCES hotels(id)"); err != nil {
		return err
	}
	log.Println("房间表创建成功")
	return nil
}

// InitRecordTable 初始化訂單表
func InitRecordTable() error {
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
		booking_num INT NOT NULL DEFAULT 1, -- 訂房間數
		payment_type VARCHAR(45) NOT NULL DEFAULT 'cash', -- 付款方式
		payment_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- 付款狀態：pending / paid / failed / refunded
		status BOOLEAN NOT NULL DEFAULT TRUE,
		create_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_records_user FOREIGN KEY (user_id) REFERENCES users(id),
		CONSTRAINT fk_records_hotel FOREIGN KEY (hotel_id) REFERENCES hotels(id),
		CONSTRAINT fk_records_room FOREIGN KEY (room_id) REFERENCES rooms(id)
	  );
	  CREATE INDEX IF NOT EXISTS idx_records_id ON records(id);
	  CREATE INDEX IF NOT EXISTS idx_records_payment_status ON records(payment_status); -- 待付款列表
	  CREATE INDEX IF NOT EXISTS idx_records_user_payment ON records(user_id, payment_status); -- 依使用者查待付款
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Printf("创建訂單表失败: %v", err)
		return err
	}
	// 既有表補上入住時間。先可空回填，再改為必填，避免舊資料擋住 NOT NULL
	if _, err := db.Exec(`ALTER TABLE records ADD COLUMN IF NOT EXISTS checkin_time TIMESTAMP`); err != nil {
		log.Printf("添加入住時間欄位失敗: %v", err)
		return err
	}
	if _, err := db.Exec(`UPDATE records SET checkin_time = booking_time WHERE checkin_time IS NULL`); err != nil {
		log.Printf("回填入住時間失敗: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ALTER COLUMN checkin_time SET NOT NULL`); err != nil {
		log.Printf("設定入住時間為必填失敗: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ADD COLUMN IF NOT EXISTS booking_num INT`); err != nil {
		log.Printf("添加訂房間數欄位失敗: %v", err)
		return err
	}
	if _, err := db.Exec(`UPDATE records SET booking_num = 1 WHERE booking_num IS NULL`); err != nil {
		log.Printf("回填訂房間數失敗: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ALTER COLUMN booking_num SET DEFAULT 1`); err != nil {
		log.Printf("設定訂房間數預設值失敗: %v", err)
		return err
	}
	if _, err := db.Exec(`ALTER TABLE records ALTER COLUMN booking_num SET NOT NULL`); err != nil {
		log.Printf("設定訂房間數為必填失敗: %v", err)
		return err
	}
	// 既有表（當初無 FK）補上外鍵
	if err := ensureForeignKey("records", "fk_records_user", "FOREIGN KEY (user_id) REFERENCES users(id)"); err != nil {
		return err
	}
	if err := ensureForeignKey("records", "fk_records_hotel", "FOREIGN KEY (hotel_id) REFERENCES hotels(id)"); err != nil {
		return err
	}
	if err := ensureForeignKey("records", "fk_records_room", "FOREIGN KEY (room_id) REFERENCES rooms(id)"); err != nil {
		return err
	}
	log.Println("訂單表创建成功")
	return nil
}
