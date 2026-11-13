package models

type WalletBalance struct {
	WalletID           string
	Balance            int64
	Currency           string
	RecentTransactions []PostingItem
}
