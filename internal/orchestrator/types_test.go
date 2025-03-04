package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewManager(t *testing.T) {
	cases := []struct {
		name           string
		expectedQueLen int
	}{
		{
			name:           "Create new manager",
			expectedQueLen: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()

			assert.NotNil(t, manager)
			_, err := manager.GetExpressions()
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedQueLen, len(manager.TasksQueue.storage))
		})
	}
}

func TestStoreExpression(t *testing.T) {
	cases := []struct {
		name       string
		id         uint32
		expression Expression
	}{
		{
			name: "Store simple expression",
			id:   1,
			expression: Expression{
				id:         1,
				operator:   "+",
				status:     "waiting",
				leftValue:  make(chan float64, 1),
				rightValue: make(chan float64, 1),
			},
		},
		{
			name: "Store expression with parent",
			id:   2,
			expression: Expression{
				id:         2,
				parentId:   1,
				rootId:     1,
				operator:   "*",
				status:     "waiting",
				childSide:  "left",
				leftValue:  make(chan float64, 1),
				rightValue: make(chan float64, 1),
			},
		},
		{
			name: "Store root expression",
			id:   3,
			expression: Expression{
				id:         3,
				isRoot:     true,
				rootId:     3,
				operator:   "/",
				status:     "waiting",
				childSide:  "nil",
				leftValue:  make(chan float64, 1),
				rightValue: make(chan float64, 1),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()

			manager.StoreExpression(tc.id, tc.expression)

			value, ok := manager.AllExpressions.Load(tc.id)
			assert.True(t, ok)
			storedExpr := value.(Expression)
			assert.Equal(t, tc.expression.id, storedExpr.id)
			assert.Equal(t, tc.expression.operator, storedExpr.operator)
			assert.Equal(t, tc.expression.status, storedExpr.status)
		})
	}
}

func TestGetExpressions(t *testing.T) {
	cases := []struct {
		name                string
		storedExpressions   map[uint32]Expression
		expectedExpressions int
		expectedError       error
	}{
		{
			name:                "Empty expressions",
			storedExpressions:   map[uint32]Expression{},
			expectedExpressions: 0,
			expectedError:       nil,
		},
		{
			name: "Multiple expressions",
			storedExpressions: map[uint32]Expression{
				1: {
					id:       1,
					operator: "+",
					status:   "waiting",
				},
				2: {
					id:       2,
					operator: "-",
					status:   "waiting",
				},
				3: {
					id:       3,
					operator: "*",
					status:   "working",
				},
			},
			expectedExpressions: 3,
			expectedError:       nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			expressions, err := manager.GetExpressions()

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedExpressions, len(expressions))
		})
	}
}

func TestGetExpressionById(t *testing.T) {
	cases := []struct {
		name              string
		storedExpressions map[uint32]Expression
		idToGet           uint32
		expectedError     error
		shouldExist       bool
	}{
		{
			name:              "Expression not found",
			storedExpressions: map[uint32]Expression{},
			idToGet:           1,
			expectedError:     ErrExpressionNotFound,
			shouldExist:       false,
		},
		{
			name: "Expression found",
			storedExpressions: map[uint32]Expression{
				1: {
					id:       1,
					operator: "+",
					status:   "waiting",
				},
			},
			idToGet:       1,
			expectedError: nil,
			shouldExist:   true,
		},
		{
			name: "One of multiple expressions",
			storedExpressions: map[uint32]Expression{
				1: {
					id:       1,
					operator: "+",
					status:   "waiting",
				},
				2: {
					id:       2,
					operator: "-",
					status:   "working",
				},
			},
			idToGet:       2,
			expectedError: nil,
			shouldExist:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			expression, err := manager.GetExpressionById(tc.idToGet)

			assert.Equal(t, tc.expectedError, err)
			if tc.shouldExist {
				assert.Equal(t, tc.idToGet, expression.id)
				assert.Equal(t, tc.storedExpressions[tc.idToGet].operator, expression.operator)
				assert.Equal(t, tc.storedExpressions[tc.idToGet].status, expression.status)
			}
		})
	}
}

func TestGetParentID(t *testing.T) {
	cases := []struct {
		name              string
		storedExpressions map[uint32]Expression
		idToGet           uint32
		expectedParentID  uint32
		expectedError     error
	}{
		{
			name:              "Expression not found",
			storedExpressions: map[uint32]Expression{},
			idToGet:           1,
			expectedParentID:  0,
			expectedError:     ErrExpressionNotFound,
		},
		{
			name: "Root expression (no parent)",
			storedExpressions: map[uint32]Expression{
				1: {
					id:       1,
					isRoot:   true,
					parentId: 0,
					status:   "waiting",
				},
			},
			idToGet:          1,
			expectedParentID: 0,
			expectedError:    nil,
		},
		{
			name: "Expression with parent",
			storedExpressions: map[uint32]Expression{
				1: {
					id:       1,
					isRoot:   true,
					parentId: 0,
					status:   "waiting",
				},
				2: {
					id:       2,
					parentId: 1,
					rootId:   1,
					status:   "waiting",
				},
			},
			idToGet:          2,
			expectedParentID: 1,
			expectedError:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			parentID, err := manager.GetParentID(tc.idToGet)

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedParentID, parentID)
		})
	}
}

func TestGetRootId(t *testing.T) {
	cases := []struct {
		name              string
		storedExpressions map[uint32]Expression
		idToGet           uint32
		expectedRootID    uint32
		expectedError     error
	}{
		{
			name:              "Expression not found",
			storedExpressions: map[uint32]Expression{},
			idToGet:           1,
			expectedRootID:    0,
			expectedError:     ErrExpressionNotFound,
		},
		{
			name: "Root expression",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					isRoot: true,
					rootId: 1,
					status: "waiting",
				},
			},
			idToGet:        1,
			expectedRootID: 1,
			expectedError:  nil,
		},
		{
			name: "Expression with different root",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					isRoot: true,
					rootId: 1,
					status: "waiting",
				},
				2: {
					id:       2,
					parentId: 1,
					rootId:   1,
					status:   "waiting",
				},
				3: {
					id:       3,
					parentId: 2,
					rootId:   1,
					status:   "waiting",
				},
			},
			idToGet:        3,
			expectedRootID: 1,
			expectedError:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			rootID, err := manager.GetRootId(tc.idToGet)

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedRootID, rootID)
		})
	}
}

func TestGetNodePosition(t *testing.T) {
	cases := []struct {
		name              string
		storedExpressions map[uint32]Expression
		idToGet           uint32
		expectedPosition  string
		expectedError     error
	}{
		{
			name:              "Expression not found",
			storedExpressions: map[uint32]Expression{},
			idToGet:           1,
			expectedPosition:  "",
			expectedError:     ErrExpressionNotFound,
		},
		{
			name: "Root node position",
			storedExpressions: map[uint32]Expression{
				1: {
					id:        1,
					isRoot:    true,
					childSide: "nil",
					status:    "waiting",
				},
			},
			idToGet:          1,
			expectedPosition: "nil",
			expectedError:    nil,
		},
		{
			name: "Left node position",
			storedExpressions: map[uint32]Expression{
				1: {
					id:        1,
					isRoot:    true,
					childSide: "nil",
					status:    "waiting",
				},
				2: {
					id:        2,
					parentId:  1,
					rootId:    1,
					childSide: "left",
					status:    "waiting",
				},
			},
			idToGet:          2,
			expectedPosition: "left",
			expectedError:    nil,
		},
		{
			name: "Right node position",
			storedExpressions: map[uint32]Expression{
				1: {
					id:        1,
					isRoot:    true,
					childSide: "nil",
					status:    "waiting",
				},
				2: {
					id:        2,
					parentId:  1,
					rootId:    1,
					childSide: "right",
					status:    "waiting",
				},
			},
			idToGet:          2,
			expectedPosition: "right",
			expectedError:    nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			position, err := manager.GetNodePosition(tc.idToGet)

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedPosition, position)
		})
	}
}

func TestUpdateExpressionResult(t *testing.T) {
	cases := []struct {
		name              string
		storedExpressions map[uint32]Expression
		idToUpdate        uint32
		resultValue       float64
		initialStatus     string
		expectedStatus    string
		expectedError     error
	}{
		{
			name:              "Expression not found",
			storedExpressions: map[uint32]Expression{},
			idToUpdate:        1,
			resultValue:       42.0,
			expectedError:     ErrExpressionNotFound,
		},
		{
			name: "Update result for waiting expression",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					status: "waiting",
					result: 0.0,
				},
			},
			idToUpdate:     1,
			resultValue:    42.0,
			initialStatus:  "waiting",
			expectedStatus: "done",
			expectedError:  nil,
		},
		{
			name: "Update result for working expression",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					status: "working",
					result: 0.0,
				},
			},
			idToUpdate:     1,
			resultValue:    42.0,
			initialStatus:  "working",
			expectedStatus: "done",
			expectedError:  nil,
		},
		{
			name: "Update result for done expression",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					status: "done",
					result: 10.0,
				},
			},
			idToUpdate:     1,
			resultValue:    42.0,
			initialStatus:  "done",
			expectedStatus: "done",
			expectedError:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			err := manager.UpdateExpressionResult(tc.idToUpdate, tc.resultValue)

			assert.Equal(t, tc.expectedError, err)

			if err == nil {
				expr, _ := manager.GetExpressionById(tc.idToUpdate)
				assert.Equal(t, tc.resultValue, expr.result)

				if tc.initialStatus == "working" {
					assert.Equal(t, "done", expr.status)
				} else {
					assert.Equal(t, tc.initialStatus, expr.status)
				}
			}
		})
	}
}

func TestIsRoot(t *testing.T) {
	cases := []struct {
		name              string
		storedExpressions map[uint32]Expression
		idToCheck         uint32
		expectedIsRoot    bool
		expectedError     error
	}{
		{
			name:              "Expression not found",
			storedExpressions: map[uint32]Expression{},
			idToCheck:         1,
			expectedIsRoot:    false,
			expectedError:     ErrExpressionNotFound,
		},
		{
			name: "Root expression",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					isRoot: true,
					status: "waiting",
				},
			},
			idToCheck:      1,
			expectedIsRoot: true,
			expectedError:  nil,
		},
		{
			name: "Non-root expression",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					isRoot: true,
					status: "waiting",
				},
				2: {
					id:       2,
					isRoot:   false,
					parentId: 1,
					rootId:   1,
					status:   "waiting",
				},
			},
			idToCheck:      2,
			expectedIsRoot: false,
			expectedError:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			isRoot, err := manager.IsRoot(tc.idToCheck)

			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedIsRoot, isRoot)
		})
	}
}

func TestUpdateExpressionValue(t *testing.T) {
	cases := []struct {
		name              string
		storedExpressions map[uint32]Expression
		idToUpdate        uint32
		valuePosition     string
		value             float64
		expectedError     error
	}{
		{
			name:              "Expression not found",
			storedExpressions: map[uint32]Expression{},
			idToUpdate:        1,
			valuePosition:     "left",
			value:             42.0,
			expectedError:     ErrExpressionNotFound,
		},
		{
			name: "Update left value",
			storedExpressions: map[uint32]Expression{
				1: {
					id:         1,
					status:     "waiting",
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				},
			},
			idToUpdate:    1,
			valuePosition: "left",
			value:         42.0,
			expectedError: nil,
		},
		{
			name: "Update right value",
			storedExpressions: map[uint32]Expression{
				1: {
					id:         1,
					status:     "waiting",
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				},
			},
			idToUpdate:    1,
			valuePosition: "right",
			value:         42.0,
			expectedError: nil,
		},
		{
			name: "Update nil position (root node)",
			storedExpressions: map[uint32]Expression{
				1: {
					id:         1,
					isRoot:     true,
					status:     "waiting",
					childSide:  "nil",
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				},
			},
			idToUpdate:    1,
			valuePosition: "nil",
			value:         42.0,
			expectedError: nil,
		},
		{
			name: "Invalid node position",
			storedExpressions: map[uint32]Expression{
				1: {
					id:         1,
					status:     "waiting",
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				},
			},
			idToUpdate:    1,
			valuePosition: "invalid",
			value:         42.0,
			expectedError: ErrInvalidNodePosition,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			err := manager.UpdateExpressionValue(tc.idToUpdate, tc.valuePosition, tc.value)

			assert.Equal(t, tc.expectedError, err)

			if err == nil && tc.valuePosition != "nil" {
				expr, _ := manager.GetExpressionById(tc.idToUpdate)

				if tc.valuePosition == "left" && len(expr.leftValue) > 0 {
					val := <-expr.leftValue
					assert.Equal(t, tc.value, val)
				} else if tc.valuePosition == "right" && len(expr.rightValue) > 0 {
					val := <-expr.rightValue
					assert.Equal(t, tc.value, val)
				}
			}
		})
	}

	t.Run("Wrong left channel. expression is root", func(t *testing.T) {
		manager := NewManager()

		expr := Expression{
			id:        1,
			parentId:  0,
			isRoot:    true,
			leftValue: make(chan float64, 1),
		}
		manager.StoreExpression(1, expr)
		expr.leftValue <- 42.0

		err := manager.UpdateExpressionValue(1, "left", 42.0)
		assert.Equal(t, ErrInvalidChannelCondition, err)
	})
	t.Run("Wrong left channel. expression is not root", func(t *testing.T) {
		manager := NewManager()

		expr := Expression{
			id:        1,
			parentId:  2,
			isRoot:    false,
			leftValue: make(chan float64, 1),
		}
		manager.StoreExpression(1, expr)
		expr.leftValue <- 42.0

		err := manager.UpdateExpressionValue(1, "left", 42.0)
		assert.Equal(t, ErrInvalidChannelCondition, err)
	})
	t.Run("Wrong right channel", func(t *testing.T) {
		manager := NewManager()

		expr := Expression{
			id:         1,
			parentId:   0,
			isRoot:     true,
			rightValue: make(chan float64, 1),
		}
		manager.StoreExpression(1, expr)
		expr.rightValue <- 42.0

		err := manager.UpdateExpressionValue(1, "right", 42.0)
		assert.Equal(t, ErrInvalidChannelCondition, err)
	})
	t.Run("Wrong right channel. expression is not root", func(t *testing.T) {
		manager := NewManager()

		expr := Expression{
			id:         1,
			parentId:   2,
			isRoot:     false,
			rightValue: make(chan float64, 1),
		}
		manager.StoreExpression(1, expr)
		expr.rightValue <- 42.0

		err := manager.UpdateExpressionValue(1, "right", 42.0)
		assert.Equal(t, ErrInvalidChannelCondition, err)
	})
}
func TestUpdateExpressionStatus(t *testing.T) {
	cases := []struct {
		name              string
		storedExpressions map[uint32]Expression
		idToUpdate        uint32
		newStatus         string
		expectedError     error
	}{
		{
			name:              "Expression not found",
			storedExpressions: map[uint32]Expression{},
			idToUpdate:        1,
			newStatus:         "working",
			expectedError:     ErrExpressionNotFound,
		},
		{
			name: "Update status from waiting to working",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					status: "waiting",
				},
			},
			idToUpdate:    1,
			newStatus:     "working",
			expectedError: nil,
		},
		{
			name: "Update status from working to done",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					status: "working",
				},
			},
			idToUpdate:    1,
			newStatus:     "done",
			expectedError: nil,
		},
		{
			name: "Update status of a root expression",
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					isRoot: true,
					status: "waiting",
				},
			},
			idToUpdate:    1,
			newStatus:     "working",
			expectedError: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()
			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			err := manager.UpdateExpressionStatus(tc.idToUpdate, tc.newStatus)
			assert.Equal(t, tc.expectedError, err)

			if err == nil {
				expr, _ := manager.GetExpressionById(tc.idToUpdate)
				assert.Equal(t, tc.newStatus, expr.status)
			}
		})
	}
}

func TestAddTask(t *testing.T) {
	cases := []struct {
		name              string
		initialQueueState []uint32
		taskToAdd         uint32
		expectedQueueSize int
	}{
		{
			name:              "Add task to empty queue",
			initialQueueState: []uint32{},
			taskToAdd:         1,
			expectedQueueSize: 1,
		},
		{
			name:              "Add task to non-empty queue",
			initialQueueState: []uint32{1, 2, 3},
			taskToAdd:         4,
			expectedQueueSize: 4,
		},
		{
			name:              "Add duplicate task",
			initialQueueState: []uint32{1, 2, 3},
			taskToAdd:         2,
			expectedQueueSize: 4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()

			for _, taskId := range tc.initialQueueState {
				manager.AddTask(taskId)
			}

			assert.Equal(t, len(tc.initialQueueState), len(manager.TasksQueue.storage))

			manager.AddTask(tc.taskToAdd)

			assert.Equal(t, tc.expectedQueueSize, len(manager.TasksQueue.storage))
			if len(manager.TasksQueue.storage) > 0 {
				assert.Equal(t, tc.taskToAdd, manager.TasksQueue.storage[len(manager.TasksQueue.storage)-1])
			}
		})
	}
}

func TestNextTask(t *testing.T) {
	cases := []struct {
		name                      string
		initialQueueState         []uint32
		storedExpressions         map[uint32]Expression
		expectedTaskId            uint32
		expectedError             error
		expectedInitialStatus     string
		expectedStatusAfterUpdate string
	}{
		{
			name:              "Empty queue",
			initialQueueState: []uint32{},
			storedExpressions: map[uint32]Expression{},
			expectedTaskId:    0,
			expectedError:     ErrQueueIsEmpty,
		},
		{
			name:              "Expression not found in storage",
			initialQueueState: []uint32{1},
			storedExpressions: map[uint32]Expression{},
			expectedTaskId:    0,
			expectedError:     ErrExpressionNotFound,
		},
		{
			name:              "Get task from queue with waiting status",
			initialQueueState: []uint32{1},
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					status: "waiting",
				},
			},
			expectedTaskId:            1,
			expectedError:             nil,
			expectedInitialStatus:     "waiting",
			expectedStatusAfterUpdate: "working",
		},
		{
			name:              "Get task from queue with non-waiting status",
			initialQueueState: []uint32{1},
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					status: "done",
				},
			},
			expectedTaskId:            1,
			expectedError:             nil,
			expectedInitialStatus:     "done",
			expectedStatusAfterUpdate: "done",
		},
		{
			name:              "Multiple tasks in queue",
			initialQueueState: []uint32{1, 2, 3},
			storedExpressions: map[uint32]Expression{
				1: {
					id:     1,
					status: "waiting",
				},
				2: {
					id:     2,
					status: "waiting",
				},
				3: {
					id:     3,
					status: "waiting",
				},
			},
			expectedTaskId:            1,
			expectedError:             nil,
			expectedInitialStatus:     "waiting",
			expectedStatusAfterUpdate: "working",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manager := NewManager()

			for id, expr := range tc.storedExpressions {
				manager.StoreExpression(id, expr)
			}

			for _, taskId := range tc.initialQueueState {
				manager.AddTask(taskId)
			}

			initialQueueSize := len(manager.TasksQueue.storage)

			taskId, err := manager.NextTask()

			assert.Equal(t, tc.expectedError, err)

			if err == nil {
				assert.Equal(t, tc.expectedTaskId, taskId)

				assert.Equal(t, initialQueueSize-1, len(manager.TasksQueue.storage))

				expr, _ := manager.GetExpressionById(taskId)
				if tc.expectedInitialStatus == "waiting" {
					assert.Equal(t, tc.expectedStatusAfterUpdate, expr.status)
				} else {
					assert.Equal(t, tc.expectedInitialStatus, expr.status)
				}
			}
		})
	}
}
