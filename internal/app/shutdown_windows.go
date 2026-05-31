//go:build windows

package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

const shutdownTimeout = 3 * time.Second

func (a *application) beginShutdown(reason string, exitAfterGraceful bool) {
	if a == nil {
		return
	}
	done := a.startShutdown(reason)
	a.shutdownWatchdog.Do(func() {
		go func() {
			select {
			case <-done:
				logShutdownEvent("process exit: graceful shutdown complete")
				if exitAfterGraceful {
					os.Exit(0)
				}
				if a.hwnd != 0 && a.winapi.postMessage != nil {
					if a.winapi.postMessage(a.hwnd, wmShutdownComplete, 0, 0) {
						return
					}
				}
				os.Exit(0)
			case <-time.After(shutdownTimeout):
				logShutdownEvent("shutdown timeout: exceeded %s", shutdownTimeout)
				logShutdownStackDump()
				logShutdownEvent("process exit: forced after shutdown timeout")
				os.Exit(0)
			}
		}()
	})
}

func (a *application) startShutdown(reason string) <-chan struct{} {
	if a == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	a.shutdownOnce.Do(func() {
		a.shuttingDown.Store(true)
		go func() {
			defer close(a.shutdownDone)
			a.performShutdown(reason)
		}()
	})
	return a.shutdownDone
}

func (a *application) performShutdown(reason string) {
	logShutdownEvent("Shutdown start: reason=%s", reason)
	if a.cancel != nil {
		a.cancel()
	}
	if a.signalStop != nil {
		a.signalStop()
	}

	a.unregisterHooks()

	handles := requestRuntimeRunnersStop(a.runner, a.clicker)
	if len(handles) == 0 {
		logShutdownEvent("worker stop requested: no active runtime workers")
	}
	for _, handle := range handles {
		if handle.requested {
			logShutdownEvent("worker stop requested: %s", handle.name)
		} else {
			logShutdownEvent("worker stop already in progress: %s", handle.name)
		}
	}

	logShutdownEvent("input release requested")
	releaseInjectedInputs()
	releaseMouseButtons()
	logShutdownEvent("input released")

	waitShutdownRuntimeStops(handles)
	a.runtimeInputTarget.Store(0)
	logShutdownEvent("Shutdown graceful complete")
}

func (a *application) unregisterHooks() {
	if a == nil {
		return
	}
	var keyboardHook uintptr
	var mouseHook uintptr
	a.cleanupMu.Lock()
	if a.hook != 0 {
		keyboardHook = a.hook
		a.hook = 0
	}
	if a.mouseHook != 0 {
		mouseHook = a.mouseHook
		a.mouseHook = 0
	}
	a.cleanupMu.Unlock()

	if keyboardHook != 0 {
		a.winapi.unhookWindowsHook(keyboardHook)
		logShutdownEvent("hook unregistered: keyboard")
	}
	if mouseHook != 0 {
		a.winapi.unhookWindowsHook(mouseHook)
		logShutdownEvent("hook unregistered: mouse")
	}
}

func waitShutdownRuntimeStops(handles []runtimeStopHandle) {
	var wg sync.WaitGroup
	for _, handle := range handles {
		handle := handle
		wg.Add(1)
		go func() {
			defer wg.Done()
			if handle.done != nil {
				<-handle.done
			}
			handle.releaseActive()
			logShutdownEvent("worker stopped: %s", handle.name)
		}()
	}
	wg.Wait()
}

func logShutdownStackDump() {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	logShutdownEvent("goroutine stack dump:\n%s", string(buf[:n]))
}

func logShutdownEvent(format string, args ...any) {
	path := shutdownLogPath()
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer file.Close()
	message := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(file, "%s %s\n", time.Now().Format(time.RFC3339Nano), message)
}

func shutdownLogPath() string {
	dir := os.TempDir()
	if dir == "" {
		return "diablo-helper-shutdown.log"
	}
	return filepath.Join(dir, "diablo-helper-shutdown.log")
}
