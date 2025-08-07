package atomicarray

import "sync/atomic"

type AtomicArray[T any] struct {
	arr atomic.Pointer[[]T]
}

func New[T any](arr *[]T) *AtomicArray[T] {
	s := &AtomicArray[T]{}
	s.arr.Store(arr)
	return s
}

func (s *AtomicArray[T]) Load() []T {
	return *s.arr.Load()
}

func (s *AtomicArray[T]) Store(newSlice []T) {
	s.arr.Store(&newSlice)
}

func (s *AtomicArray[T]) Update(updateFunc func(arr []T) []T) {
	for {
		arr := s.arr.Load()

		res := updateFunc(*arr)
		if res == nil {
			return
		}

		if s.arr.CompareAndSwap(arr, &res) {
			return
		}
	}
}
