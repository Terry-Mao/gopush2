package skiplist

import (
	"strconv"
	"testing"
)

func TestSkipListNew(t *testing.T) {
	sl := New()

	for i := 0; i < MaxLevel; i++ {
		if sl.Head.forward[i] != nil {
			t.Error("skiplist head init error")
		}
	}

	if sl.Level != 0 {
		t.Error("skiplist level init error")
	}

	if sl.Length != 0 {
		t.Error("skiplist Length init error")
	}
}

func TestSkipListInsert(t *testing.T) {
	sl := New()
	err := sl.Insert(5, "test5", 5)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(5, "testxx", 5)
	if err != ErrNodeExists {
		t.Error("node must exists")
	}

	err = sl.Insert(3, "test3", 3)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(7, "test7", 7)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(9, "test9", 9)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(1, "test1", 1)
	if err != nil {
		t.Error(err)
	}

	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	i, j := 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		if n.Expire != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}
}

func TestSkipListUpdate(t *testing.T) {
	sl := New()
	err := sl.Insert(5, "test5", 5)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(5, "testxx", 5)
	if err != ErrNodeExists {
		t.Error("node must exists")
	}

	err = sl.Insert(3, "test3", 3)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(7, "test7", 7)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(9, "test9", 9)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(1, "test1", 1)
	if err != nil {
		t.Error(err)
	}

	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	i, j := 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		if n.Expire != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}

	sl.Update(1, "test111", 2)
	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	n := sl.Equal(1)
	if n == nil {
		t.Error("skiplist update error")
	}

	if n.Member != "test111" {
		t.Error("skiplist update error")
	}

	if n.Expire != 2 {
		t.Error("skiplist update error")
	}
}

func TestSkipListDelete(t *testing.T) {
	sl := New()
	err := sl.Insert(5, "test5", 5)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(5, "testxx", 5)
	if err != ErrNodeExists {
		t.Error("node must exists")
	}

	err = sl.Insert(3, "test3", 3)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(7, "test7", 7)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(9, "test9", 9)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(1, "test1", 1)
	if err != nil {
		t.Error(err)
	}

	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	i, j := 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		if n.Expire != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}

	sl.Update(1, "test111", 2)
	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	n := sl.Equal(1)
	if n == nil {
		t.Error("skiplist update error")
	}

	if n.Member != "test111" {
		t.Error("skiplist update error")
	}

	if n.Expire != 2 {
		t.Error("skiplist update error")
	}

	err = sl.Delete(10)
	if err != ErrNodeNotExists {
		t.Error("skiplist node delete error")
	}

	err = sl.Delete(9)
	if err != nil {
		t.Error(err)
	}

	if sl.Length != 4 {
		t.Error("skiplist Length init 3")
	}

	sl.Update(1, "test1", 1)
	i, j = 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		if n.Expire != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}
}

func TestSkipListGreate(t *testing.T) {
	sl := New()
	err := sl.Insert(5, "test5", 5)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(5, "testxx", 5)
	if err != ErrNodeExists {
		t.Error("node must exists")
	}

	n := sl.Greate(5)
	if n != nil {
		t.Error("skiplist greate error")
	}

	n = sl.Greate(4)
	if n == nil || n.Score != 5 || n.Member != "test5" {
		t.Error("skiplist greate error")
	}

	err = sl.Insert(3, "test3", 3)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(7, "test7", 7)
	if err != nil {
		t.Error(err)
	}

	n = sl.Greate(7)
	if n != nil {
		t.Error("skiplist greate error")
	}

	err = sl.Insert(9, "test9", 9)
	if err != nil {
		t.Error(err)
	}

	n = sl.Greate(7)
	if n == nil || n.Score != 9 || n.Member != "test9" {
		t.Error("skiplist greate error")
	}

	err = sl.Insert(1, "test1", 1)
	if err != nil {
		t.Error(err)
	}

	n = sl.Greate(1)
	if n == nil || n.Score != 3 || n.Member != "test3" {
		t.Error("skiplist greate error")
	}

	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	i, j := 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		if n.Expire != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}

	sl.Update(1, "test111", 1)
	n = sl.Equal(1)
	if n == nil || n.Member != "test111" || n.Score != 1 {
		t.Error("skiplist find node error")
	}

	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	err = sl.Delete(10)
	if err != ErrNodeNotExists {
		t.Error("skiplist node delete error")
	}

	err = sl.Delete(9)
	if err != nil {
		t.Error(err)
	}

	if sl.Length != 4 {
		t.Error("skiplist Length init 3")
	}

	n = sl.Equal(9)
	if n != nil {
		t.Error("skiplist find node error")
	}

	if sl.Length != 4 {
		t.Error("skiplist Length init 3")
	}

	sl.Update(1, "test1", 1)
	n = sl.Equal(1)
	if n == nil || n.Member != "test1" || n.Score != 1 {
		t.Error("skiplist find node error")
	}

	i, j = 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		if n.Expire != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}
}

func TestSkipListEqual(t *testing.T) {
	sl := New()
	err := sl.Insert(5, "test5", 5)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(5, "testxx", 5)
	if err != ErrNodeExists {
		t.Error("node must exists")
	}

	err = sl.Insert(3, "test3", 3)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(7, "test7", 7)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(9, "test9", 9)
	if err != nil {
		t.Error(err)
	}

	err = sl.Insert(1, "test1", 1)
	if err != nil {
		t.Error(err)
	}

	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	i, j := 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		if n.Expire != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}

	sl.Update(1, "test111", 2)
	n := sl.Equal(1)
	if n == nil || n.Member != "test111" || n.Score != 1 {
		t.Error("skiplist find node error")
	}

	if sl.Length != 5 {
		t.Error("skiplist Length init 3")
	}

	err = sl.Delete(10)
	if err != ErrNodeNotExists {
		t.Error("skiplist node delete error")
	}

	err = sl.Delete(9)
	if err != nil {
		t.Error(err)
	}

	if sl.Length != 4 {
		t.Error("skiplist Length init 3")
	}

	n = sl.Equal(9)
	if n != nil {
		t.Error("skiplist find node error")
	}

	if sl.Length != 4 {
		t.Error("skiplist Length init 3")
	}

	sl.Update(1, "test1", 1)
	n = sl.Equal(1)
	if n == nil || n.Member != "test1" || n.Score != 1 {
		t.Error("skiplist find node error")
	}

	i, j = 1, int64(1)
	for n := sl.Head.Next(); n != nil; n = n.Next() {
		if n.Member != "test"+strconv.FormatInt(n.Score, 10) {
			t.Errorf("skiplist node member error (%s)", n.Member)
		}

		if n.Score != j {
			t.Error("skiplist node score error")
		}

		if n.Expire != j {
			t.Error("skiplist node score error")
		}

		i++
		j += 2
	}
}
