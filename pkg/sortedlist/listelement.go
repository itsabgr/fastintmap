package sortedlist

import (
	"github.com/itsabgr/atomic2"
	"sync/atomic"
	"unsafe"
)

// ListElement is an element of a list.
type ListElement struct {
	previousElement unsafe.Pointer // is nil for the first item in list
	nextElement     unsafe.Pointer // is nil for the last item in list
	key             atomic2.Uintptr
	value           unsafe.Pointer
	deleted         atomic2.Uintptr // marks the item as deleting or deleted
}

// NewElement returns an initialized list element.
func NewElement(key uintptr, value interface{}) *ListElement {
	return &ListElement{
		key:   atomic2.Uintptr(key),
		value: unsafe.Pointer(&value),
	}
}

// Value returns the value of the list item.
func (e *ListElement) Value() (value interface{}) {
	return *(*interface{})(atomic.LoadPointer(&e.value))
}

// Key returns the key of the list item.
func (e *ListElement) Key() uintptr {
	return uintptr(e.key)
}

// Next returns the item on the right.
func (e *ListElement) Next() *ListElement {
	return (*ListElement)(atomic.LoadPointer(&e.nextElement))
}

// Previous returns the item on the left.
func (e *ListElement) Previous() *ListElement {
	return (*ListElement)(atomic.LoadPointer(&e.previousElement))
}

// setValue sets the value of the item.
// The value needs to be wrapped in unsafe.Pointer already.
func (e *ListElement) setValue(value unsafe.Pointer) {
	atomic.StorePointer(&e.value, value)
}

// casValue compares and swaps the values of the item.
// The to value needs to be wrapped in unsafe.Pointer already.
func (e *ListElement) casValue(from interface{}, to unsafe.Pointer) bool {
	old := atomic.LoadPointer(&e.value)
	if *(*interface{})(old) != from {
		return false
	}
	return atomic.CompareAndSwapPointer(&e.value, old, to)
}
