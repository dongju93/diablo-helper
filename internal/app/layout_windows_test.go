//go:build windows

package app

import "testing"

func TestComputeLayoutKeepsDesignSizeAtMax(t *testing.T) {
	lo := computeLayout(layoutDesignW, layoutDesignH)

	if lo.leftX != layoutLX || lo.leftW != layoutLW || lo.rx != layoutRX {
		t.Fatalf("left column = x:%d w:%d rx:%d, want x:%d w:%d rx:%d", lo.leftX, lo.leftW, lo.rx, layoutLX, layoutLW, layoutRX)
	}
	if lo.rw != layoutDesignW-layoutRX-layoutMX {
		t.Fatalf("right width = %d, want %d", lo.rw, layoutDesignW-layoutRX-layoutMX)
	}
}

func TestComputeLayoutScalesDownForSmallClient(t *testing.T) {
	max := computeLayout(layoutDesignW, layoutDesignH)
	small := computeLayout(744, 561)

	if small.leftW >= max.leftW {
		t.Fatalf("small left width = %d, want less than %d", small.leftW, max.leftW)
	}
	if small.skillBtnW >= max.skillBtnW {
		t.Fatalf("small skill button width = %d, want less than %d", small.skillBtnW, max.skillBtnW)
	}
	if small.statusBarW >= max.statusBarW {
		t.Fatalf("small status width = %d, want less than %d", small.statusBarW, max.statusBarW)
	}
	if small.skillBtnW <= 0 || small.statusTextW <= 0 || small.pauseBtnW <= 0 {
		t.Fatalf("small layout has non-positive widths: skill=%d status=%d pause=%d", small.skillBtnW, small.statusTextW, small.pauseBtnW)
	}
}
