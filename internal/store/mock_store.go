package store

// MockStoreKey represents the composite key for the store.
// In this case, it's fixed to owner, project, and name, but you could
// adjust it if needed for other use cases.
type MockStoreKey struct {
	Owner   string
	Project string
	Name    string
}

// MockStore is a generic in-memory store for any value type V.
// It uses a map for efficient lookups, additions, and deletions.
// The getName function is provided at instantiation to extract the
// name (third key component) from the value, keeping the store flexible
// without requiring V to implement a specific interface.
type MockStore[K comparable, V any] struct {
	data map[K]V
}

// NewStore creates a new instance of the generic store.
// You provide a function to extract the name from the value type V.
func NewMockStore[K comparable, V any]() *MockStore[K, V] {
	return &MockStore[K, V]{
		data: make(map[K]V),
	}
}

// Get retrieves a value by its full key components.
func (s *MockStore[K, V]) Get(key K) (V, error) {
	v, exists := s.data[key]
	if !exists {
		var zero V
		return zero, ErrNotFound
	}
	return v, nil
}

// Add inserts a new value, using the provided owner/project and extracting name from the value.
func (s *MockStore[K, V]) Add(key K, value V) error {
	if _, exists := s.data[key]; exists {
		return ErrExist
	}
	s.data[key] = value
	return nil
}

func (s *MockStore[K, V]) Upsert(key K, value V) error {
	s.data[key] = value
	return nil
}

func (s *MockStore[K, V]) Update(key K, value V) error {
	if _, exists := s.data[key]; !exists {
		return ErrNotFound
	}
	s.data[key] = value
	return nil
}

// Delete removes a value, using the provided owner/project and extracting name from the value.
func (s *MockStore[K, V]) Delete(key K) error {
	if _, exists := s.data[key]; !exists {
		return ErrNotFound
	}
	delete(s.data, key)
	return nil
}

func (s *MockStore[K, V]) List() ([]V, error) {
	items := []V{}

	for _, item := range s.data {
		items = append(items, item)
	}

	return items, nil
}
