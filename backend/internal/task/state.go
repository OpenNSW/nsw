package task

// SimpleMapStateManager implements StateManager using a map.
type SimpleMapStateManager struct {
	state map[string]any
}

func NewSimpleMapStateManager(initialState map[string]any) *SimpleMapStateManager {
	if initialState == nil {
		initialState = make(map[string]any)
	}
	return &SimpleMapStateManager{state: initialState}
}

func (m *SimpleMapStateManager) Get(key string) any {
	return m.state[key]
}

func (m *SimpleMapStateManager) Set(key string, value any) {
	m.state[key] = value
}

func (m *SimpleMapStateManager) GetAll() map[string]any {
	return m.state
}
