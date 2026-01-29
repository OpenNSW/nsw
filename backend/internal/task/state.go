package task

// SimpleMapStateManager implements StateManager using a map.
type SimpleMapStateManager struct {
	state map[string]interface{}
}

func NewSimpleMapStateManager(initialState map[string]interface{}) *SimpleMapStateManager {
	if initialState == nil {
		initialState = make(map[string]interface{})
	}
	return &SimpleMapStateManager{state: initialState}
}

func (m *SimpleMapStateManager) Get(key string) (interface{}, bool) {
	val, ok := m.state[key]
	return val, ok
}

func (m *SimpleMapStateManager) Set(key string, value interface{}) error {
	m.state[key] = value
	return nil
}

func (m *SimpleMapStateManager) GetAll() map[string]interface{} {
	return m.state
}
