package ui

import "fyne.io/fyne/v2/widget"

type HistoryCard struct {
	widget.Card
	Sender          string
	TransferContent string
	ReceiveTime     int
}

func NewHistoryCard() *HistoryCard {
	return &HistoryCard{
		Card:            *widget.NewCard("", "", nil),
		Sender:          "",
		TransferContent: "",
		ReceiveTime:     0,
	}
}
