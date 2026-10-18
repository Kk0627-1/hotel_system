package hotel

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

// Hotel 單筆酒店（列表／詳情）
type Hotel struct {
	ID         string `json:"hotel_id"`
	Name       string `json:"hotel_name"`
	CreateTime string `json:"create_time"`
	Status     bool   `json:"status"`
}

type ListHotelsRequest struct {
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	HotelID    string `json:"hotel_id"`
	HotelName  string `json:"hotel_name"`
	CreateTime string `json:"create_time"` // 可選，格式 YYYY-MM-DD，按建立日期篩選
	Status     *bool  `json:"status,omitempty"`
}

type ListHotelsResult struct {
	Hotels     []Hotel `json:"hotels"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	TotalPages int     `json:"total_page"`
}

const maxListPageSize = 100

// ListHotels 分頁列表，並依條件篩選（酒店 ID、名稱模糊、建立日期、狀態）
func ListHotels(db *sql.DB, req ListHotelsRequest) (ListHotelsResult, error) {
	if req.Page < 1 {
		log.Printf("查詢酒店列表欄位驗證失敗：page 無效 page=%d", req.Page)
		return ListHotelsResult{}, fmt.Errorf("%w: page is required and must be at least 1", ErrInvalidListParams)
	}
	if req.PageSize < 1 {
		log.Printf("查詢酒店列表欄位驗證失敗：page_size 無效 page_size=%d", req.PageSize)
		return ListHotelsResult{}, fmt.Errorf("%w: page_size is required and must be at least 1", ErrInvalidListParams)
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
	log.Printf("查詢酒店列表：page=%d page_size=%d hotel_id=%q hotel_name=%q create_time=%q status=%s",
		page, pageSize, strings.TrimSpace(req.HotelID), strings.TrimSpace(req.HotelName), strings.TrimSpace(req.CreateTime), statusLog)

	var conds []string
	var args []interface{}
	argN := 1

	if id := strings.TrimSpace(req.HotelID); id != "" {
		conds = append(conds, fmt.Sprintf("id = $%d", argN))
		args = append(args, id)
		argN++
	}
	if name := strings.TrimSpace(req.HotelName); name != "" {
		conds = append(conds, fmt.Sprintf("name ILIKE $%d", argN))
		args = append(args, "%"+escapeILikePattern(name)+"%")
		argN++
	}
	if ct := strings.TrimSpace(req.CreateTime); ct != "" {
		_, err := time.Parse("2006-01-02", ct)
		if err != nil {
			log.Printf("查詢酒店列表欄位驗證失敗：create_time 格式錯誤 create_time=%q", ct)
			return ListHotelsResult{}, fmt.Errorf("%w: create_time must be YYYY-MM-DD", ErrInvalidListParams)
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

	countQuery := "SELECT COUNT(*) FROM hotels " + whereClause
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		log.Printf("查詢酒店列表筆數失敗：err=%v", err)
		return ListHotelsResult{}, err
	}

	offset := (page - 1) * pageSize
	listArgs := append(append([]interface{}{}, args...), pageSize, offset)
	limitArg := argN
	offsetArg := argN + 1
	selectQuery := fmt.Sprintf(
		`SELECT id, name, create_time, status FROM hotels %s ORDER BY create_time DESC LIMIT $%d OFFSET $%d`,
		whereClause, limitArg, offsetArg,
	)

	rows, err := db.Query(selectQuery, listArgs...)
	if err != nil {
		log.Printf("查詢酒店列表失敗：err=%v", err)
		return ListHotelsResult{}, err
	}
	defer rows.Close()

	var hotels []Hotel
	for rows.Next() {
		var h Hotel
		var id int
		var ct time.Time
		if err := rows.Scan(&id, &h.Name, &ct, &h.Status); err != nil {
			log.Printf("讀取酒店列表資料失敗：err=%v", err)
			return ListHotelsResult{}, err
		}
		h.ID = strconv.Itoa(id)
		h.CreateTime = ct.Format(time.RFC3339)
		hotels = append(hotels, h)
	}
	if err := rows.Err(); err != nil {
		log.Printf("遍歷酒店列表失敗：err=%v", err)
		return ListHotelsResult{}, err
	}

	totalPages := 0
	if total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	log.Printf("查詢酒店列表成功：total=%d page=%d page_size=%d count=%d", total, page, pageSize, len(hotels))

	return ListHotelsResult{
		Hotels:     hotels,
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
