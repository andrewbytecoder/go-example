package buffer

import (
	"testing"
)

func TestRingBuffer_ForEach(t *testing.T) {
	// 测试空缓冲区
	t.Run("EmptyBuffer", func(t *testing.T) {
		rb := NewRingBuffer[int](8)
		count := 0
		rb.ForEach(func(v *int) bool {
			count++
			return true
		})
		if count != 0 {
			t.Errorf("Expected 0 iterations, got %d", count)
		}
	})

	// 测试连续数据
	t.Run("ContiguousData", func(t *testing.T) {
		rb := NewRingBuffer[int](8)
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)

		expected := []int{1, 2, 3}
		index := 0
		rb.ForEach(func(v *int) bool {
			if *v != expected[index] {
				t.Errorf("Expected %d, got %d", expected[index], *v)
			}
			index++
			return true
		})
	})

	// 测试环形数据
	t.Run("WrappedData", func(t *testing.T) {
		rb := NewRingBuffer[int](4)
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)
		rb.Push(4) // 缓冲区满，触发增长
		rb.Pop()   // 弹出一个元素，使数据变为环形
		rb.Push(5) // 新元素插入到头部

		expected := []int{2, 3, 4, 5}
		index := 0
		rb.ForEach(func(v *int) bool {
			if *v != expected[index] {
				t.Errorf("Expected %d, got %d", expected[index], *v)
			}
			index++
			return true
		})
	})

	// 测试提前终止
	t.Run("EarlyTermination", func(t *testing.T) {
		rb := NewRingBuffer[int](8)
		rb.Push(1)
		rb.Push(2)
		rb.Push(3)

		count := 0
		rb.ForEach(func(v *int) bool {
			count++
			return count < 2 // 只处理前两个元素
		})
		if count != 2 {
			t.Errorf("Expected 2 iterations, got %d", count)
		}
	})
}

// segment defines a KCP segment
type segment struct {
	conv     uint32
	cmd      uint8
	frg      uint8
	wnd      uint16
	ts       uint32
	sn       uint32
	una      uint32
	rto      uint32
	xmit     uint32
	resendts uint32
	fastack  uint32
	acked    uint32 // mark if the seg has acked
	data     []byte
}

func TestRingBuffer_Range(t *testing.T) {

	// 测试环形数据
	t.Run("WrappedData", func(t *testing.T) {

		rcv_queue := NewRingBuffer[segment](120 * 2)

		length := 0

		for seg := range rcv_queue.ForEach {
			length += len(seg.data)
			if seg.frg == 0 {
				break
			}
		}

	})

}
