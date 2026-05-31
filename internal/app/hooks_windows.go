//go:build windows

package app

import (
	"runtime"
	"unsafe"
)

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
			bounds := appInstance.currentWindowBounds(hwnd)
			info.MaxPosition.X = bounds.maxPositionX
			info.MaxPosition.Y = bounds.maxPositionY
			info.MaxSize.X = bounds.maximizedW
			info.MaxSize.Y = bounds.maximizedH
			info.MinTrackSize.X = bounds.minW
			info.MinTrackSize.Y = bounds.minH
			info.MaxTrackSize.X = bounds.maxW
			info.MaxTrackSize.Y = bounds.maxH
		}
		return 0
	case wmDpiChanged:
		appInstance.handleDPIChanged(hwnd, wParam, lParam)
		return 0
	case wmPaint:
		appInstance.paint(hwnd)
		return 0
	case wmEraseBkgnd:
		return 1
	case wmCtlColorStatic:
		return appInstance.colorStatic(wParam, uintptr(unsafe.Pointer(lParam)))
	case wmCtlColorBtn:
		return appInstance.colorStatic(wParam, uintptr(unsafe.Pointer(lParam)))
	case wmCtlColorEdit:
		return appInstance.colorEdit(wParam)
	case wmDrawItem:
		appInstance.drawButton((*drawItemStruct)(lParam))
		return 1
	case wmCommand:
		if appInstance.handleCommand(wParam) {
			return 0
		}
	case wmRunnerError:
		if appInstance.shuttingDown.Load() {
			return 0
		}
		appInstance.showPendingRunnerError()
		return 0
	case wmRuntimeStopComplete:
		appInstance.finishAsyncRuntimeStop()
		return 0
	case wmClose:
		appInstance.setStatus("종료 중입니다. 실행 중인 입력 작업을 정리합니다.")
		appInstance.beginShutdown("WM_CLOSE", false)
		return 0
	case wmQueryEndSession:
		appInstance.beginShutdown("WM_QUERYENDSESSION", true)
		return 1
	case wmEndSession:
		if wParam != 0 {
			appInstance.beginShutdown("WM_ENDSESSION", true)
		}
		return 0
	case wmShutdownComplete:
		appInstance.winapi.destroyWindow(hwnd)
		return 0
	case wmDestroy:
		app := appInstance
		if !app.shuttingDown.Load() {
			app.beginShutdown("WM_DESTROY", true)
		}
		app.cleanup()
		app.winapi.postQuitMessage(0)
		return 0
	}
	return defWindowProc(hwnd, msg, wParam, lParam)
}

func lowLevelKeyboardProc(code int, wParam uintptr, lParam unsafe.Pointer) uintptr {
	if code < 0 || appInstance == nil || appInstance.shuttingDown.Load() || lParam == nil {
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

	vk := uint16(event.VKCode)
	handled := appInstance.handleKeyEvent(vk, keyDown)
	clearHookKeyState(&vk, &keyDown)
	if handled {
		return 1
	}
	return callNextHookEx(code, wParam, lParam)
}

func lowLevelMouseProc(code int, wParam uintptr, lParam unsafe.Pointer) uintptr {
	if code < 0 || appInstance == nil || appInstance.shuttingDown.Load() || lParam == nil {
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
		clearHookKeyState(&vk, &down)
		return callNextHookEx(code, wParam, lParam)
	}
	handled := appInstance.handleKeyEvent(vk, down)
	clearHookKeyState(&vk, &down)
	if handled {
		return 1
	}
	return callNextHookEx(code, wParam, lParam)
}

//go:noinline
func clearHookKeyState(vk *uint16, down *bool) {
	*vk = 0
	*down = false
	runtime.KeepAlive(vk)
	runtime.KeepAlive(down)
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
