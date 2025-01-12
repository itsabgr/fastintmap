package sortedlist

import (
	"github.com/itsabgr/go-handy"
	"sync/atomic"
	"unsafe"
)

// List is a sorted doubly linked list.
type List struct {
	_noCopy handy.NoCopy
	count   uintptr
	head    *ListElement
}

// New returns an initialized list.
func New() *List {
	return &List{head: &ListElement{}}
}

// Len returns the number of elements within the list.
func (l *List) Len() int {
	if l == nil { // not initialized yet?
		return 0
	}

	return int(atomic.LoadUintptr(&l.count))
}

// Head returns the head item of the list.
func (l *List) Head() *ListElement {
	if l == nil { // not initialized yet?
		return nil
	}

	return l.head
}

// First returns the first item of the list.
func (l *List) First() *ListElement {
	if l == nil { // not initialized yet?
		return nil
	}

	return l.head.Next()
}

// Add adds an item to the list and returns false if an item for the hash existed.
// searchStart = nil will start to search at the head item
func (l *List) Add(element *ListElement, searchStart *ListElement) (existed bool, inserted bool) {
	left, found, right := l.search(searchStart, element)
	if found != nil { // existing item found
		return true, false
	}

	return false, l.insertAt(element, left, right)
}

// AddOrUpdate adds or updates an item to the list.
func (l *List) AddOrUpdate(element *ListElement, searchStart *ListElement) bool {
	left, found, right := l.search(searchStart, element)
	if found != nil { // existing item found
		found.setValue(element.value) // update the value
		return true
	}

	return l.insertAt(element, left, right)
}

// Cas compares and swaps the value of an item in the list.
func (l *List) Cas(element *ListElement, oldValue interface{}, searchStart *ListElement) bool {
	_, found, _ := l.search(searchStart, element)
	if found == nil { // no existing item found
		return false
	}

	if found.casValue(oldValue, element.value) {
		return true
	}
	return false
}

func (l *List) search(searchStart *ListElement, item *ListElement) (left *ListElement, found *ListElement, right *ListElement) {
	if searchStart != nil && item.key < searchStart.key { // key would remain left from item? {
		searchStart = nil // start search at head
	}

	if searchStart == nil { // start search at head?
		left = l.head
		found = left.Next()
		if found == nil { // no items beside head?
			return nil, nil, nil
		}
	} else {
		found = searchStart
	}

	for {
		if item.key == found.key { // key already exists
			return nil, found, nil
		}

		if item.key < found.key { // new item needs to be inserted before the found value
			if l.head == left {
				return nil, nil, found
			}
			return left, nil, found
		}

		// go to next element in sorted linked list
		left = found
		found = left.Next()
		if found == nil { // no more items on the right
			return left, nil, nil
		}
	}
}

func (l *List) insertAt(element *ListElement, left *ListElement, right *ListElement) bool {
	if left == nil {
		//element->previous = head
		element.previousElement = unsafe.Pointer(l.head)
		//element->next = right
		element.nextElement = unsafe.Pointer(right)

		// insert at head, head-->next = element
		if !atomic.CompareAndSwapPointer(&l.head.nextElement, unsafe.Pointer(right), unsafe.Pointer(element)) {
			return false // item was modified concurrently
		}

		//right->previous = element
		if right != nil {
			if !atomic.CompareAndSwapPointer(&right.previousElement, unsafe.Pointer(l.head), unsafe.Pointer(element)) {
				return false // item was modified concurrently
			}
		}
	} else {
		element.previousElement = unsafe.Pointer(left)
		element.nextElement = unsafe.Pointer(right)

		if !atomic.CompareAndSwapPointer(&left.nextElement, unsafe.Pointer(right), unsafe.Pointer(element)) {
			return false // item was modified concurrently
		}

		if right != nil {
			if !atomic.CompareAndSwapPointer(&right.previousElement, unsafe.Pointer(left), unsafe.Pointer(element)) {
				return false // item was modified concurrently
			}
		}
	}

	atomic.AddUintptr(&l.count, 1)
	return true
}

// Delete deletes an element from the list.
func (l *List) Delete(element *ListElement) {
	if !element.deleted.CAS(0, 1) {
		return // concurrent delete of the item in progress
	}

	for {
		left := element.Previous()
		right := element.Next()

		if left == nil { // element is first item in list?
			if !atomic.CompareAndSwapPointer(&l.head.nextElement, unsafe.Pointer(element), unsafe.Pointer(right)) {
				continue // now head item was inserted concurrently
			}
		} else {
			if !atomic.CompareAndSwapPointer(&left.nextElement, unsafe.Pointer(element), unsafe.Pointer(right)) {
				continue // item was modified concurrently
			}
		}
		if right != nil {
			atomic.CompareAndSwapPointer(&right.previousElement, unsafe.Pointer(element), unsafe.Pointer(left))
		}
		break
	}

	atomic.AddUintptr(&l.count, ^uintptr(0)) // decrease counter
}
