package core

import (
	"sync"

	"github.com/leonard-atorough/castrum/ecs"
)

type ListNode struct {
	Prev  *ListNode
	Next  *ListNode
	Key   string
	value any
}

type QueryCache struct {
	Capacity int
	Cache    map[string]*ListNode
	Head     *ListNode
	Tail     *ListNode
	mu       sync.Mutex
}

// the query cache is an LRU cache used to store the results of queries to avoid repeated computation.
// We'll implement the cache following these principles:
// 1. The cache is a fixed-sized, doubly linked list that stores the results of queries.
// 2. The cache will provide a way (either via method or public field) to allow cache invalidation when the underlying data changes.
// 3. The cache will implement remaining cache methods.
// 4. The cache will be thread-safe, allowing concurrent access from multiple goroutines.
func NewQueryCache(capacity int) *QueryCache {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}

	return &QueryCache{
		Capacity: capacity,
		Cache:    make(map[string]*ListNode),
	}
}

func (c *QueryCache) Get(key string) ([]ecs.EntityID, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.Cache[key]; ok {
		c.moveToHead(node)
		return node.value.([]ecs.EntityID), true
	}
	return nil, false
}

func (c *QueryCache) Set(key string, value []ecs.EntityID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.Cache[key]; ok {
		node.value = value
		c.moveToHead(node)
	} else {
		if len(c.Cache) >= c.Capacity {
			delete(c.Cache, c.Tail.Key)
			c.removeNode(c.Tail)
		}
		newNode := &ListNode{
			Key:   key,
			value: value,
		}
		c.addToHead(newNode)
		c.Cache[key] = newNode
	}
}

func (c *QueryCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if node, ok := c.Cache[key]; ok {
		c.removeNode(node)
		delete(c.Cache, key)
	}
}

func (c *QueryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Head = nil
	c.Tail = nil
	c.Cache = make(map[string]*ListNode)
}

func (c *QueryCache) addToHead(node *ListNode) {
	if c.Head == nil {
		c.Head = node
		c.Tail = node
	} else {
		node.Next = c.Head
		c.Head.Prev = node
		c.Head = node
	}
}

func (c *QueryCache) moveToHead(node *ListNode) {
	if node == c.Head {
		return
	}

	c.removeNode(node)
	c.addToHead(node)
}

func (c *QueryCache) removeNode(node *ListNode) {
	if node.Prev != nil {
		node.Prev.Next = node.Next
	} else {
		c.Head = node.Next
	}
	if node.Next != nil {
		node.Next.Prev = node.Prev
	}
	if node == c.Tail {
		c.Tail = node.Prev
	}
	node.Prev = nil
	node.Next = nil
}
