package manager

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"

	"github.com/OpenNSW/nsw/internal/task/container"
)

// cacheNode represents a node in the LRU cache doubly linked list
type cacheNode struct {
	taskID    uuid.UUID
	container *container.Container
	prev      *cacheNode
	next      *cacheNode
}

// containerCache is a fixed-length LRU cache for storing active containers
type containerCache struct {
	capacity int
	cache    map[uuid.UUID]*cacheNode
	head     *cacheNode // Most recently used
	tail     *cacheNode // Least recently used
	mu       sync.RWMutex
}

// newContainerCache creates a new LRU cache with the specified capacity
func newContainerCache(capacity int) *containerCache {
	if capacity <= 0 {
		capacity = 100 // Default capacity
	}
	return &containerCache{
		capacity: capacity,
		cache:    make(map[uuid.UUID]*cacheNode),
	}
}

// Get retrieves a container from cache and marks it as recently used
func (c *containerCache) Get(taskID uuid.UUID) (*container.Container, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.cache[taskID]
	if !exists {
		return nil, false
	}

	// Move to front (most recently used)
	c.moveToFront(node)
	return node.container, true
}

// Set adds or updates a container in the cache
func (c *containerCache) Set(taskID uuid.UUID, cont *container.Container) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already exists, update and move to front
	if node, exists := c.cache[taskID]; exists {
		node.container = cont
		c.moveToFront(node)
		return
	}

	// Create new node
	newNode := &cacheNode{
		taskID:    taskID,
		container: cont,
	}

	// Add to cache map and front of list
	c.cache[taskID] = newNode
	c.addToFront(newNode)

	// Evict least recently used if over capacity
	if len(c.cache) > c.capacity {
		c.evictLRU()
	}
}

// Delete removes a container from the cache
func (c *containerCache) Delete(taskID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()

	node, exists := c.cache[taskID]
	if !exists {
		return
	}

	c.removeNode(node)
	delete(c.cache, taskID)
}

// Clear removes all entries from the cache
func (c *containerCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[uuid.UUID]*cacheNode)
	c.head = nil
	c.tail = nil
}

// Len returns the current number of items in the cache
func (c *containerCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// moveToFront moves a node to the front of the list (most recently used)
func (c *containerCache) moveToFront(node *cacheNode) {
	if c.head == node {
		return
	}
	c.removeNode(node)
	c.addToFront(node)
}

// addToFront adds a node to the front of the list
func (c *containerCache) addToFront(node *cacheNode) {
	node.next = c.head
	node.prev = nil

	if c.head != nil {
		c.head.prev = node
	}
	c.head = node

	if c.tail == nil {
		c.tail = node
	}
}

// removeNode removes a node from the list
func (c *containerCache) removeNode(node *cacheNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		c.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		c.tail = node.prev
	}
}

// evictLRU removes the least recently used item from the cache
func (c *containerCache) evictLRU() {
	if c.tail == nil {
		return
	}

	lruNode := c.tail
	c.removeNode(lruNode)
	delete(c.cache, lruNode.taskID)

	slog.Debug("evicted container from cache",
		"taskID", lruNode.taskID,
		"cacheSize", len(c.cache))
}
