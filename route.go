package main

import (
	"net/http"
)

func (app *Config) routes() http.Handler {
	mux := http.NewServeMux()
	// 用戶相關路由
	// user
	// 註冊
	mux.Handle("/register", RegisterHandler{})
	// 登入
	mux.Handle("/login", LoginHandler{})

	// 飯店相關路由
	// hotel
	// 創建飯店
	mux.Handle("/hotel/create", CreateHotelHandler{})
	// 更新飯店
	mux.Handle("/hotel/update", UpdateHotelHandler{})
	// 刪除飯店
	mux.Handle("/hotel/delete", DeleteHotelHandler{})
	// 查詢飯店列表
	mux.Handle("/hotel/list", ListHotelsHandler{})

	// 房間相關路由
	// room

	// 創建房間
	mux.Handle("/room/create", CreateRoomHandler{})
	// 更新房間
	mux.Handle("/room/update", UpdateRoomHandler{})
	// 刪除房間
	mux.Handle("/room/delete", DeleteRoomHandler{})
	// 查詢房間列表
	mux.Handle("/room/list", ListRoomsHandler{})

	// 訂單相關路由
	// record
	// 創建訂房
	mux.Handle("/reservation/create", CreateReservationHandler{})

	// 支付相關路由
	// 帳單報表(已支付)
	mux.Handle("/payment/list", ListPurchasedBillsHandler{})
	// 確認支付
	mux.Handle("/payment/confirm", ConfirmPaymentHandler{})
	// 取消支付(訂單)
	mux.Handle("/payment/cancel", CancelPaymentHandler{})
	// 帳單報表(已支付)
	mux.Handle("/payment/bill", BillReportHandler{})
	// 待付款列表
	mux.Handle("/payment/pending", ListPendingPaymentsHandler{})

	// 錢包相關路由
	// wallet
	// 錢包儲值列表
	mux.Handle("/wallet/recharge", WalletRechargeHandler{})

	return mux
}
