package main

import "github.com/getlantern/systray"

type menuPool struct {
	// read-only
	_menu []*systray.MenuItem
	_c    chan int
	_len  int
}

func newMenuPool(size int) *menuPool {
	menu := menuPool{
		_menu: make([]*systray.MenuItem, size),
		_c:    make(chan int, 1),
	}
	for idx := range menu._menu {
		menu._menu[idx] = systray.AddMenuItem("", "")
		menu._menu[idx].Hide()

		go func(id int, ch chan struct{}) {
			for range ch {
				menu._c <- id
			}
		}(idx, menu._menu[idx].ClickedCh)
	}
	menu._len = len(menu._menu)
	return &menu
}

func (m *menuPool) WaitSignal() <-chan int {
	return m._c
}

func (m *menuPool) GetSize() int {
	return m._len
}

func (m *menuPool) HideAll() {
	for _, item := range m._menu {
		item.Hide()
	}
}

func (m *menuPool) UpdateTitle(index int, title string, andShow bool) bool {
	if index >= m._len {
		return false
	}
	m._menu[index].SetTitle(title)
	if andShow {
		m._menu[index].Show()
	}
	return true
}
