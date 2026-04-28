//go:build windows

package app

import "unsafe"

func wndProc(hwnd uintptr, msg uint32, wParam uintptr, lParam unsafe.Pointer) uintptr {
	if appInstance == nil {
		return defWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case wmCreate:
		appInstance.hwnd = hwnd
		appInstance.createControls(hwnd)
		return 0
	case wmSize:
		if appInstance.controls.status != 0 {
			appInstance.repositionControls()
		}
		return 0
	case wmGetMinMaxInfo:
		if lParam != nil {
			info := (*minMaxInfo)(lParam)
			info.MinTrackSize.X = windowMinW
			info.MinTrackSize.Y = windowMinH
			info.MaxSize.X = windowMaxW
			info.MaxSize.Y = windowMaxH
			info.MaxTrackSize.X = windowMaxW
			info.MaxTrackSize.Y = windowMaxH
		}
		return 0
	case wmPaint:
		appInstance.paint(hwnd)
		return 0
	case wmEraseBkgnd:
		return 1
	case wmCtlColorStatic:
		return appInstance.colorStatic(wParam)
	case wmCtlColorBtn:
		return appInstance.colorStatic(wParam)
	case wmCtlColorEdit:
		return appInstance.colorEdit(wParam)
	case wmDrawItem:
		appInstance.drawButton((*drawItemStruct)(lParam))
		return 1
	case wmCommand:
		if appInstance.handleCommand(wParam) {
			return 0
		}
	case wmClose:
		procDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		if appInstance.hook != 0 {
			procUnhookWindowsHook.Call(appInstance.hook)
			appInstance.hook = 0
		}
		if appInstance.mouseHook != 0 {
			procUnhookWindowsHook.Call(appInstance.mouseHook)
			appInstance.mouseHook = 0
		}
		appInstance.runner.Stop()
		appInstance.clicker.Stop()
		appInstance.disposeUIResources()
		procPostQuitMessage.Call(0)
		return 0
	}
	return defWindowProc(hwnd, msg, wParam, lParam)
}

func lowLevelKeyboardProc(code int, wParam uintptr, lParam unsafe.Pointer) uintptr {
	if code < 0 || appInstance == nil || lParam == nil {
		return callNextHookEx(code, wParam, lParam)
	}

	keyDown := false
	switch wParam {
	case wmKeyDown, wmSysKeyDown:
		keyDown = true
	case wmKeyUp, wmSysKeyUp:
	default:
		return callNextHookEx(code, wParam, lParam)
	}

	event := (*keyboardHookStruct)(lParam)
	if event.Flags&llkhfInjected != 0 {
		return callNextHookEx(code, wParam, lParam)
	}

	if appInstance.handleKeyEvent(uint16(event.VKCode), keyDown) {
		return 1
	}
	return callNextHookEx(code, wParam, lParam)
}

func lowLevelMouseProc(code int, wParam uintptr, lParam unsafe.Pointer) uintptr {
	if code < 0 || appInstance == nil || lParam == nil {
		return callNextHookEx(code, wParam, lParam)
	}

	if !isMouseHookMessage(wParam) {
		return callNextHookEx(code, wParam, lParam)
	}

	event := (*mouseHookStruct)(lParam)
	if event.Flags&llmhfInjected != 0 {
		return callNextHookEx(code, wParam, lParam)
	}

	vk, down, ok := mouseEventKey(wParam, event)
	if !ok {
		return callNextHookEx(code, wParam, lParam)
	}
	if appInstance.handleKeyEvent(vk, down) {
		return 1
	}
	return callNextHookEx(code, wParam, lParam)
}

func isMouseHookMessage(wParam uintptr) bool {
	switch wParam {
	case wmLButtonDown, wmLButtonUp, wmRButtonDown, wmRButtonUp,
		wmMButtonDown, wmMButtonUp, wmXButtonDown, wmXButtonUp:
		return true
	}
	return false
}

func mouseEventKey(wParam uintptr, event *mouseHookStruct) (uint16, bool, bool) {
	switch wParam {
	case wmLButtonDown:
		return vkLButton, true, true
	case wmLButtonUp:
		return vkLButton, false, true
	case wmRButtonDown:
		return vkRButton, true, true
	case wmRButtonUp:
		return vkRButton, false, true
	case wmMButtonDown:
		return vkMButton, true, true
	case wmMButtonUp:
		return vkMButton, false, true
	case wmXButtonDown, wmXButtonUp:
		button := event.MouseData >> 16
		switch button {
		case xButton1:
			return vkXButton1, wParam == wmXButtonDown, true
		case xButton2:
			return vkXButton2, wParam == wmXButtonDown, true
		}
	}
	return 0, false, false
}
