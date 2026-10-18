package room

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

// ErrInvalidListParams 列表查詢參數不合法（例如日期格式）
var ErrInvalidListParams = errors.New("invalid list parameters")

type Room struct {
	ID         string `json:"room_id"`
	HotelID    string `json:"hotel_id"`
	RoomType   string `json:"room_type"`
	Price      int    `json:"price"`
	Quantity   int    `json:"quantity"`
	Status     bool   `json:"status"`
	CreateTime string `json:"create_time"`
}

type ListRoomsRequest struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	RoomID     string `json:"room_id"`
	HotelID    string `json:"hotel_id"`
	RoomType   string `json:"room_type"`
	Price      int    `json:"price"`
	Status     *bool  `json:"status,omitempty"`
	CreateTime string `json:"create_time"` // 可選，格式 YYYY-MM-DD，按建立日期篩選
}

type ListRoomsResult struct {
	Rooms      []Room `json:"rooms"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_page"`
}

const maxListPageSize = 100

// ListRooms 分頁列表，並依條件篩選（房間 ID、號碼模糊、類型、價格、狀態、建立日期）
func ListRooms(db *sql.DB, req ListRoomsRequest) (ListRoomsResult, error) {
	if req.Page < 1 {
		log.Printf("查詢房間列表欄位驗證失敗：page 無效 page=%d", req.Page)
		return ListRoomsResult{}, fmt.Errorf("%w: page is required and must be at least 1", ErrInvalidListParams)
	}
	if req.PageSize < 1 {
		log.Printf("查詢房間列表欄位驗證失敗：page_size 無效 page_size=%d", req.PageSize)
		return ListRoomsResult{}, fmt.Errorf("%w: page_size is required and must be at least 1", ErrInvalidListParams)
	}

	page := req.Page
	pageSize := req.PageSize
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}

	statusLog := "nil"
	if req.Status != nil {
		statusLog = strconv.FormatBool(*req.Status)
	}
	log.Printf("查詢房間列表：page=%d page_size=%d room_id=%q hotel_id=%q room_type=%q price=%d create_time=%q status=%s",
		page, pageSize, strings.TrimSpace(req.RoomID), strings.TrimSpace(req.HotelID), strings.TrimSpace(req.RoomType), req.Price, strings.TrimSpace(req.CreateTime), statusLog)

	var conds []string
	var args []interface{}
	argN := 1

	if id := strings.TrimSpace(req.RoomID); id != "" {
		conds = append(conds, fmt.Sprintf("id = $%d", argN))
		args = append(args, id)
		argN++
	}
	if hotelID := strings.TrimSpace(req.HotelID); hotelID != "" {
		conds = append(conds, fmt.Sprintf("hotel_id = $%d", argN))
		args = append(args, hotelID)
		argN++
	}
	if roomType := strings.TrimSpace(req.RoomType); roomType != "" {
		conds = append(conds, fmt.Sprintf("room_type ILIKE $%d", argN))
		args = append(args, "%"+escapeILikePattern(roomType)+"%")
		argN++
	}
	if req.Price > 0 {
		conds = append(conds, fmt.Sprintf("price = $%d", argN))
		args = append(args, req.Price)
		argN++
	}
	if ct := strings.TrimSpace(req.CreateTime); ct != "" {
		if _, err := time.Parse("2006-01-02", ct); err != nil {
			log.Printf("查詢房間列表欄位驗證失敗：create_time 格式錯誤 create_time=%q", ct)
			return ListRoomsResult{}, fmt.Errorf("%w: create_time must be YYYY-MM-DD", ErrInvalidListParams)
		}
		conds = append(conds, fmt.Sprintf("create_time::date = $%d::date", argN))
		args = append(args, ct)
		argN++
	}
	if req.Status != nil {
		conds = append(conds, fmt.Sprintf("status = $%d", argN))
		args = append(args, *req.Status)
		argN++
	}

	whereClause := ""
	if len(conds) > 0 {
		whereClause = "WHERE " + strings.Join(conds, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM rooms " + whereClause
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("查詢房間列表筆數失敗：err=%v", err)
		return ListRoomsResult{}, err
	}

	offset := (page - 1) * pageSize
	listArgs := append(append([]interface{}{}, args...), pageSize, offset)
	limitArg := argN
	offsetArg := argN + 1
	selectQuery := fmt.Sprintf(
		`SELECT id, hotel_id, room_type, price, quantity, status, create_time FROM rooms %s ORDER BY create_time DESC LIMIT $%d OFFSET $%d`,
		whereClause, limitArg, offsetArg,
	)

	rows, err := db.Query(selectQuery, listArgs...)
	if err != nil {
		log.Printf("查詢房間列表失敗：err=%v", err)
		return ListRoomsResult{}, err
	}
	defer rows.Close()

	var rooms []Room
	for rows.Next() {
		var room Room
		var id, hotelID, quantity int
		var price float64
		var ct time.Time
		if err := rows.Scan(&id, &hotelID, &room.RoomType, &price, &quantity, &room.Status, &ct); err != nil {
			log.Printf("讀取房間列表資料失敗：err=%v", err)
			return ListRoomsResult{}, err
		}
		room.ID = strconv.Itoa(id)
		room.HotelID = strconv.Itoa(hotelID)
		room.Price = int(price)
		room.Quantity = quantity
		room.CreateTime = ct.Format(time.RFC3339)
		rooms = append(rooms, room)
	}
	if err := rows.Err(); err != nil {
		log.Printf("遍歷房間列表失敗：err=%v", err)
		return ListRoomsResult{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	log.Printf("查詢房間列表成功：total=%d page=%d page_size=%d count=%d", total, page, pageSize, len(rooms))

	return ListRoomsResult{
		Rooms:      rooms,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func escapeILikePattern(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}
