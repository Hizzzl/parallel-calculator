package orchestrator

import (
	"errors"
	"sync"
)

// Errors
var (
	ErrQueueIsEmpty            = errors.New("queue is empty")
	ErrExpressionNotFound      = errors.New("expression not found")
	ErrInvalidAST              = errors.New("invalid AST")
	ErrLiteralNotFound         = errors.New("literal not found")
	ErrParentNotFound          = errors.New("parent expression not found")
	ErrInvalidNodePosition     = errors.New("invalid node position")
	ErrInvalidChannelCondition = errors.New("invalid channel condition")
	ErrInvalidExpression       = errors.New("invalid expression")
	ErrOnlyOneLiteral          = errors.New("only one literal allowed")
	ErrInvalidParentId         = errors.New("invalid parent id")
)

type Order struct {
	id          uint32
	orderNumber int
}

type ExecutionPlan struct {
	RootId      uint32
	OrderIds    []Order
	Expressions []Expression
}

type Expression struct {
	id         uint32
	parentId   uint32
	rootId     uint32
	childSide  string // Позиция узла относительно родительского узла. "left" - левый, "right" - правый, "nil" - корневой
	isRoot     bool   // true - выражение является корневым
	leftValue  chan float64
	rightValue chan float64
	operator   string
	status     string
	result     float64
}

type Queue struct {
	mu      sync.Mutex
	storage []uint32
}

type Manager struct {
	AllExpressions sync.Map // map[id]Expression
	TasksQueue     Queue
}

func (m *Manager) StoreExpression(id uint32, expression Expression) {
	m.AllExpressions.Store(id, expression)
}

func (m *Manager) GetExpressions() ([]Expression, error) {
	expressions := []Expression{}
	m.AllExpressions.Range(func(key, value interface{}) bool {
		expressions = append(expressions, value.(Expression))
		return true
	})
	return expressions, nil
}

func (m *Manager) GetExpressionById(id uint32) (Expression, error) {
	value, ok := m.AllExpressions.Load(id)
	if !ok {
		return Expression{}, ErrExpressionNotFound
	}
	return value.(Expression), nil
}

func (m *Manager) GetParentID(id uint32) (uint32, error) {
	value, ok := m.AllExpressions.Load(id)
	if !ok {
		return 0, ErrExpressionNotFound
	}
	expression := value.(Expression)
	return expression.parentId, nil
}

func (m *Manager) GetRootId(id uint32) (uint32, error) {
	value, ok := m.AllExpressions.Load(id)
	if !ok {
		return 0, ErrExpressionNotFound
	}
	expression := value.(Expression)
	return expression.rootId, nil
}

func (m *Manager) GetNodePosition(id uint32) (string, error) {
	value, ok := m.AllExpressions.Load(id)
	if !ok {
		return "", ErrExpressionNotFound
	}
	expression := value.(Expression)
	return expression.childSide, nil
}

func (m *Manager) UpdateExpressionResult(id uint32, result float64) error {
	value, ok := m.AllExpressions.Load(id)
	if !ok {
		return ErrExpressionNotFound
	}
	expression := value.(Expression)
	expression.result = result
	if expression.status == "working" {
		expression.status = "done"
	}
	m.AllExpressions.Store(id, expression)

	return nil
}

func (m *Manager) IsRoot(id uint32) (bool, error) {
	value, ok := m.AllExpressions.Load(id)
	if !ok {
		return false, ErrExpressionNotFound
	}
	expression := value.(Expression)
	return expression.isRoot, nil
}

func (m *Manager) UpdateExpressionValue(id uint32, value_position string, value float64) error {
	node, ok := m.AllExpressions.Load(id)
	if !ok {
		return ErrExpressionNotFound
	}
	expression := node.(Expression)
	switch value_position {
	case "left":
		select {
		case expression.leftValue <- value:
		default:
			return ErrInvalidChannelCondition
		}
	case "right":
		select {
		case expression.rightValue <- value:
		default:
			return ErrInvalidChannelCondition
		}
	case "nil":
		return nil
	default:
		return ErrInvalidNodePosition
	}

	m.AllExpressions.Store(id, expression)
	return nil
}

func (m *Manager) UpdateExpressionStatus(id uint32, status string) error {
	value, ok := m.AllExpressions.Load(id)
	if !ok {
		return ErrExpressionNotFound
	}
	expression := value.(Expression)
	expression.status = status
	m.AllExpressions.Store(id, expression)
	return nil
}

func (m *Manager) AddTask(taskId uint32) {
	m.TasksQueue.mu.Lock()
	defer m.TasksQueue.mu.Unlock()
	m.TasksQueue.storage = append(m.TasksQueue.storage, taskId)
}

func (m *Manager) NextTask() (uint32, error) {
	m.TasksQueue.mu.Lock()
	defer m.TasksQueue.mu.Unlock()
	if len(m.TasksQueue.storage) == 0 {
		return 0, ErrQueueIsEmpty
	}

	taskId := m.TasksQueue.storage[0]
	m.TasksQueue.storage = m.TasksQueue.storage[1:]

	taskValue, ok := m.AllExpressions.Load(taskId)
	if !ok {
		return 0, ErrExpressionNotFound
	}

	taskExpression := taskValue.(Expression)
	if taskExpression.status == "waiting" {
		taskExpression.status = "working"
	}
	m.AllExpressions.Store(taskId, taskExpression)
	return taskId, nil
}

func NewManager() *Manager {
	return &Manager{
		AllExpressions: sync.Map{},
		TasksQueue:     Queue{storage: []uint32{}},
	}
}

var ManagerInstance *Manager = NewManager()
