package main

import (
	"air_clipboard/discovery"
	"air_clipboard/models"
	"air_clipboard/transfer"
	"air_clipboard/ui"
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"go.uber.org/zap"
)

const (
	DiscoveryPort = 9456
	TransferPort  = 9457
)

// todo 获取局域网内自己的Ip
var (
	sugaredLogger *zap.SugaredLogger
	meta          = fyne.AppMetadata{
		ID:      "air_clipboard",
		Name:    "air_clipboard",
		Version: "0.0.1",
		Build:   1,
		Release: false,
		Custom:  map[string]string{},
	}
	selfInfo = &models.EndPoint{Name: "henry", DeviceName: "windows"}

	endpoints = []interface{}{selfInfo}
)

func main() {

	// backend start begin--------------------------------------------------------------------------------
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("new logger failed, err=%s", err))
	}
	defer logger.Sync()
	sugaredLogger = logger.Sugar()

	discoveryService := discovery.New(sugaredLogger, DiscoveryPort, 1, selfInfo)
	go discoveryService.Start()

	transferService := transfer.New(sugaredLogger, TransferPort, selfInfo)
	go transferService.Start()
	// backend start end --------------------------------------------------------------------------------

	myApp := app.NewWithID(meta.ID)
	mainWindow := myApp.NewWindow("air clipboard")
	mainWindow.Resize(fyne.Size{
		Width:  560,
		Height: 480,
	})
	mainWindow.CenterOnScreen()

	initShortCut(mainWindow)
	toolBar := initToolBar(mainWindow)

	endpointsVO := binding.BindUntypedList(&endpoints) // VO: view object, 供视图展示使用的对象
	go func() {
		for {
			select {
			case event := <-discoveryService.OnDiscoverEvent():
				{
					// notify transfer service
					transferService.AddTransfer(event.Endpoint)
					// update ui
					endpoints = []interface{}{selfInfo}
					endpointsVO.Set(endpoints) // 重置
					for _, point := range discoveryService.EndPoints().ToBuiltIn() {
						endpointsVO.Append(point)
					}
				}
			}
		}
	}()
	leftMenu := widget.NewListWithData(
		endpointsVO,
		func() fyne.CanvasObject {
			// 创建一个Item，其类型是Label.
			// text的长度将决定了每个Item的最小宽度
			return widget.NewLabel("template template")
		},
		func(item binding.DataItem, object fyne.CanvasObject) {
			// 将object强转为Label类型，因为在上面创建的是Label类型
			o, err := item.(binding.Untyped).Get()
			if err != nil {
				return
			}
			endpoint := o.(*models.EndPoint)
			displayText := fmt.Sprintf("%s-%s", endpoint.Name, endpoint.DeviceName)
			object.(*widget.Label).SetText(displayText)
		},
	)

	historyVO := binding.NewUntypedList()
	go func() {
		for {
			select {
			case packet := <-transferService.RecvFrom():
				{
					sendTime := time.Unix(packet.Header.SendTime, 0).Format("2006-01-02 15:04:05")
					card := ui.NewHistoryCard()
					card.SetTitle(packet.Body.Content)
					card.SetSubTitle(fmt.Sprintf("发送人：%s，时间：%s", packet.Header.Sender, sendTime))
					historyVO.Append(card)
				}
			}
		}
	}()
	transferHistoryList := widget.NewListWithData(
		historyVO,
		func() fyne.CanvasObject {
			return ui.NewHistoryCard()
		},
		func(item binding.DataItem, object fyne.CanvasObject) {
			o, err := item.(binding.Untyped).Get()
			if err != nil {
				return
			}
			msg := o.(*transfer.BaseMessage)
			historyCard := object.(*ui.HistoryCard)
			historyCard.SetTitle(string(msg.Content))
			historyCard.SetSubTitle(fmt.Sprintf("发送人：%s", msg.Sender))
		},
	)

	inputArea := widget.NewMultiLineEntry()
	inputArea.SetPlaceHolder("input something to transfer...")
	inputArea.SetMinRowsVisible(5)
	sendButton := widget.NewButton("Send", func() {
		inputText := inputArea.Text
		log.Printf("input = %s", inputText)
		inputArea.SetText("")

		transferService.Broadcast(inputText)
	})

	inputButtonGroup := container.NewHBox(layout.NewSpacer(), sendButton)
	inputGroup := container.NewVBox(inputArea, inputButtonGroup)
	mainWidget := container.NewVBox(transferHistoryList, layout.NewSpacer(), inputGroup)
	content := container.NewBorder(toolBar, nil, leftMenu, nil, mainWidget)

	mainWindow.SetContent(content)
	mainWindow.ShowAndRun()
}

func initToolBar(window fyne.Window) *widget.Toolbar {
	toolBar := widget.NewToolbar(
		widget.NewToolbarAction(theme.FolderOpenIcon(), func() {
			log.Println("folder open icon")
		}),
		widget.NewToolbarAction(theme.SearchIcon(), func() {
			log.Println("search icon")
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.SettingsIcon(), func() {
			log.Println("settings icon")
		}),
	)
	return toolBar
}

func initShortCut(window fyne.Window) {
	// todo 目前只能获取文本内容，图片内容无法获取
	shortCutCopy := &fyne.ShortcutCopy{Clipboard: window.Clipboard()}
	shortCutPaste := &fyne.ShortcutPaste{Clipboard: window.Clipboard()}
	window.Canvas().AddShortcut(shortCutCopy, func(shortcut fyne.Shortcut) {
		log.Printf("tapped Ctrl+C")
	})
	window.Canvas().AddShortcut(shortCutPaste, func(shortcut fyne.Shortcut) {
		str := window.Clipboard().Content()
		log.Printf("tapped Ctrl+V, content = %v", str)
	})
}
