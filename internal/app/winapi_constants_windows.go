//go:build windows

package app

const (
	colorWindow = 5

	cwUseDefault = 0x80000000

	wsOverlappedWindow = 0x00CF0000
	wsChild            = 0x40000000
	wsVisible          = 0x10000000
	wsTabStop          = 0x00010000

	wsExComposited = 0x02000000

	mainWindowStyle   = wsOverlappedWindow
	mainWindowExStyle = 0

	bsOwnerDraw           = 0x0000000B
	cleartypeQuality      = 5
	defaultCharset        = 1
	dwmSystemBackdropMain = 2
	esNumber              = 0x00002000
	esAutoHScroll         = 0x00000080
	transparent           = 1
	ssLeft                = 0x00000000

	dwmwaWindowCornerPref = 33
	dwmwaSystemBackdrop   = 38
	dwmwcpRound           = 2
	dtCenter              = 0x00000001
	dtVCenter             = 0x00000004
	dtSingleLine          = 0x00000020
	dtEndEllipsis         = 0x00008000
	dtNoPrefix            = 0x00000800
	ecLeftMargin          = 0x0001
	ecRightMargin         = 0x0002
	emLimitText           = 0x00C5
	emSetMargins          = 0x00D3

	// maxEditTextLen caps EM_LIMITTEXT and parse input for numeric edit controls.
	// 7 digits covers 9,999,999 ms which exceeds MaximumIntervalMS (3,600,000).
	maxEditTextLen = 7
	// maxWindowTextLen guards getWindowText against WM_SETTEXT memory DoS.
	maxWindowTextLen        = 64
	fwNormal                = 400
	fwSemiBold              = 600
	idcArrow                = 32512
	inputMouse              = 0
	inputKeyboard           = 1
	keyEventKeyUp           = 0x0002
	llkhfInjected           = 0x00000010
	llmhfInjected           = 0x00000001
	mbOK                    = 0x00000000
	mbYesNo                 = 0x00000004
	mbIconError             = 0x00000010
	mbIconWarning           = 0x00000030
	monitorDefaultToNearest = 2
	monitorDefaultToPrimary = 1
	idYes                   = 6
	mouseEventLeftDown      = 0x0002
	mouseEventLeftUp        = 0x0004
	mouseEventRightDown     = 0x0008
	mouseEventRightUp       = 0x0010
	mouseEventMiddleDown    = 0x0020
	mouseEventMiddleUp      = 0x0040
	mouseEventXDown         = 0x0080
	mouseEventXUp           = 0x0100
	logPixelsX              = 88
	smCxScreen              = 0
	smCyScreen              = 1
	swpNoZOrder             = 0x0004
	swpNoRedraw             = 0x0008
	swpNoActivate           = 0x0010
	swShowNormal            = 1
	swShowMaximized         = 3
	swShow                  = 5
	whKeyboardLL            = 13
	whMouseLL               = 14
	wpfRestoreToMaximized   = 0x0002
	wmCreate                = 0x0001
	wmDestroy               = 0x0002
	wmPaint                 = 0x000F
	wmSize                  = 0x0005
	wmGetMinMaxInfo         = 0x0024
	wmDpiChanged            = 0x02E0
	wmClose                 = 0x0010
	wmQueryEndSession       = 0x0011
	wmEndSession            = 0x0016
	wmEraseBkgnd            = 0x0014
	wmDrawItem              = 0x002B
	wmCommand               = 0x0111
	wmSetFont               = 0x0030
	wmApp                   = 0x8000
	wmRunnerError           = wmApp + 1
	wmRuntimeStopComplete   = wmApp + 2
	wmShutdownComplete      = wmApp + 3
	wmKeyDown               = 0x0100
	wmKeyUp                 = 0x0101
	wmSysKeyDown            = 0x0104
	wmSysKeyUp              = 0x0105
	wmCtlColorEdit          = 0x0133
	wmCtlColorBtn           = 0x0135
	wmCtlColorStatic        = 0x0138
	wmLButtonDown           = 0x0201
	wmLButtonUp             = 0x0202
	wmRButtonDown           = 0x0204
	wmRButtonUp             = 0x0205
	wmMButtonDown           = 0x0207
	wmMButtonUp             = 0x0208
	wmXButtonDown           = 0x020B
	wmXButtonUp             = 0x020C
	bnClicked               = 0
	odsSelected             = 0x0001
	odsDisabled             = 0x0004
	odsFocus                = 0x0010
	odsHotLight             = 0x0040
	psSolid                 = 0
	xButton1                = 0x0001
	xButton2                = 0x0002

	maxFileDialogPath = 32768

	ofnOverwritePrompt  = 0x00000002
	ofnHideReadonly     = 0x00000004
	ofnNoChangeDir      = 0x00000008
	ofnPathMustExist    = 0x00000800
	ofnFileMustExist    = 0x00001000
	ofnNoReadonlyReturn = 0x00008000
	ofnExplorer         = 0x00080000

	hkeyCurrentUser      = 0x80000001
	rrfRtRegSz           = 0x00000002
	regSz                = 1
	regOptionNonVolatile = 0
	keySetValue          = 0x0002
)
