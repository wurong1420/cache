package cache

import (
	"sync"
	"time"
)

const (
	key_header = "_sheader"
	key_tailer = "_stailer"
)

type item struct {
	key             string
	expireTimeStamp int64
	pre             *item
	next            *item
}

//---------------------------------------------------------------------------------------------------------------//
type obj struct {
	expireTimestamp int64
	value interface{}
}

type linkedList struct {
	header *item
	tailer *item
	data   map[string]obj
	lock   sync.RWMutex
}

func (l linkedList) del(key string) {
	l.lock.Lock()
	defer l.lock.Unlock()
	_, ok := l.data[key]
	if !ok {
		return
	}
	item := l.header
	for {
		next := item.next
		if next == nil || next == l.tailer {
			break
		}
		if next.key == key {
			//删除数据
			next.pre.next = next.next
			next.next.pre = next.pre
			delete(l.data, key)
			break
		}
	}
}

func (l linkedList) delItem(i *item) {
	l.lock.Lock()
	defer l.lock.Unlock()
	_, ok := l.data[i.key]
	if ok {
		delete(l.data, i.key)
	}
	i.pre.next = i.next
	i.next.pre = i.pre
}

func (l linkedList) set(key string, value interface{}, d time.Duration) {
	l.lock.Lock()
	defer l.lock.Unlock()
	expireTimestamp := time.Now().Add(d).Unix()
	o := &item{
		key:             key,
		expireTimeStamp: expireTimestamp,
	}
	e := l.header
	for {
		//可以考虑把双向链表转化为红黑树
		curr := e.next
		if curr == nil || curr == l.tailer {
			o.pre = e
			o.next = l.tailer
			e.next = o
			l.tailer.pre = e.next
			break
		}
		if curr.expireTimeStamp > expireTimestamp {
			e = curr
			continue
		}
		o.pre = curr.pre
		o.next = curr
		curr.pre.next = o
		curr.pre = o
		break
	}
	l.data[key] = obj {
		expireTimestamp: expireTimestamp,
		value: value,
	}
}

func (l linkedList) add(key string, value interface{}, d time.Duration) {
	l.lock.RLock()
	if _, ok := l.data[key]; ok {
		l.lock.RUnlock()
		return
	}
	l.lock.RUnlock()
	l.set(key, value, d)
}

func (l linkedList) get(key string) interface{} {
	l.lock.RLock()
	//需要判断数据是否过期，如果过期了就主动删除掉
	obj, ok := l.data[key]
	if ok {
		if obj.expireTimestamp < time.Now().Unix() {
			//数据已经失效了，则删除
			l.lock.RUnlock()
			l.del(key)
			return nil
		} else {
			l.lock.RUnlock()
			return obj.value
		}
	}
	l.lock.RUnlock()
	return nil
}

func (l linkedList) size() int {
	l.lock.RLock()
	defer l.lock.RUnlock()
	return len(l.data)
}

//---------------------------------------------------------------------------------------------------------------//

type janitor struct {
	segment *segment
	ticker  *time.Ticker
	stop    chan bool
}

func newJanitor(segment *segment, interval time.Duration) *janitor {
	ticker := time.NewTicker(interval)
	janitor := &janitor{
		segment: segment,
		ticker:  ticker,
	}
	go janitor.run()
	return janitor
}

func (j janitor) run() {
	for {
		select {
		case <-j.ticker.C:
		//清理过期的缓存数据
		e := j.segment.expirationLinkedList.tailer
		for {
			curr := e.pre
			if curr == nil || curr == j.segment.expirationLinkedList.header {
				break
			}
			if curr.expireTimeStamp <= time.Now().Unix() {
				e = curr.pre
				j.segment.expirationLinkedList.delItem(curr)
			}
			break
		}
		case <-j.stop:
			return
		}
	}
}

func (j janitor) drop() {
	j.stop <- true
}

//---------------------------------------------------------------------------------------------------------------//

type segment struct {
	noexpirationData     map[string]interface{}
	expirationLinkedList *linkedList
	janitor              *janitor
	lock                 sync.RWMutex
}

func newSegment(cleanInterval time.Duration) *segment {
	header := &item{
		key:  key_header,
		pre:  nil,
		next: nil,
	}
	tailer := &item{
		key:  key_tailer,
		pre:  header,
		next: header,
	}
	header.next = tailer
	header.pre = tailer

	segment := &segment{
		expirationLinkedList: &linkedList{
			header: header,
			tailer: tailer,
			data:   make(map[string]obj, 0),
		},
		noexpirationData: make(map[string]interface{}),
	}

	janitor := newJanitor(segment, cleanInterval)
	segment.janitor = janitor
	return segment
}

func (s segment) set(key string, value interface{}, d time.Duration) {
	if d == NoExpiration {
		s.lock.Lock()
		defer s.lock.Unlock()
		s.noexpirationData[key] = value
		return
	}

	s.expirationLinkedList.set(key, value, d)
}

func (s segment) add(key string, value interface{}, d time.Duration) {
	if d == NoExpiration {
		s.lock.RLock()
		if _, ok := s.noexpirationData[key]; ok {
			s.lock.RUnlock()
			return
		} else {
			s.lock.Unlock()
			s.lock.Lock()
			defer s.lock.Unlock()
			s.noexpirationData[key] = value
		}
	}
	s.expirationLinkedList.add(key, value, d)
}

func (s segment) get(key string) interface{} {
	s.lock.RLock()
	if value, ok := s.noexpirationData[key]; ok {
		s.lock.RUnlock()
		return value
	}
	s.lock.RUnlock()
	return s.expirationLinkedList.get(key)
}

func (s segment) size() int {
	s.lock.RLock()
	size := len(s.noexpirationData)
	s.lock.RUnlock()
	size += s.expirationLinkedList.size()
	return size
}