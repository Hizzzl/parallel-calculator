package orchestrator

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"parallel-calculator/internal/config"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetManager() {
	ManagerInstance = NewManager()
}

func resetConfig() {
	config.InitConfig("../../configs/.env")
}

func TestHandleCalculate(t *testing.T) {
	resetManager()
	resetConfig()

	tests := []struct {
		name               string
		requestBody        string
		expectedStatusCode int
		expectedResponse   *CalculateResponse
		setupMock          func()
	}{
		{
			name:               "valid expression",
			requestBody:        `{"expression":"2+2"}`,
			expectedStatusCode: http.StatusCreated,
			expectedResponse:   nil,
			setupMock:          func() {},
		},
		{
			name:               "invalid JSON body",
			requestBody:        `{"expression"2+2"}`,
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedResponse:   nil,
			setupMock:          func() {},
		},
		{
			name:               "invalid expression",
			requestBody:        `{"expression":"2++2"}`,
			expectedStatusCode: http.StatusUnprocessableEntity,
			expectedResponse:   nil,
			setupMock:          func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req, err := http.NewRequest(http.MethodPost, "/calculate", bytes.NewBufferString(tt.requestBody))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			HandleCalculate(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedStatusCode == http.StatusCreated {
				var response CalculateResponse
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.NotZero(t, response.ID, "Expected non-zero ID")
			}
		})
	}
}

func TestHandleGetExpressions(t *testing.T) {
	tests := []struct {
		name               string
		setupExpressions   func()
		expectedStatusCode int
		expectedLength     int
	}{
		{
			name: "empty expressions list",
			setupExpressions: func() {
				resetManager()
				resetConfig()
			},
			expectedStatusCode: http.StatusOK,
			expectedLength:     0,
		},
		{
			name: "one expression",
			setupExpressions: func() {
				resetManager()
				resetConfig()
				expr := Expression{
					id:         1,
					status:     "done",
					result:     42,
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				}
				ManagerInstance.StoreExpression(1, expr)
			},
			expectedStatusCode: http.StatusOK,
			expectedLength:     1,
		},
		{
			name: "multiple expressions",
			setupExpressions: func() {
				resetManager()
				resetConfig()
				ManagerInstance.StoreExpression(1, Expression{
					id:         1,
					status:     "done",
					result:     42,
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				})
				ManagerInstance.StoreExpression(2, Expression{
					id:         2,
					status:     "waiting",
					result:     0,
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				})
			},
			expectedStatusCode: http.StatusOK,
			expectedLength:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupExpressions()

			req, err := http.NewRequest(http.MethodGet, "/expressions", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			HandleGetExpressions(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

			var responses []ExpressionResponse
			err = json.Unmarshal(rr.Body.Bytes(), &responses)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedLength, len(responses))
		})
	}
}

func TestHandleGetExpressionByID(t *testing.T) {
	tests := []struct {
		name               string
		expressionID       string
		setupExpression    func()
		expectedStatusCode int
		expectedResponse   *ExpressionResponse
	}{
		{
			name:         "existing expression",
			expressionID: "1",
			setupExpression: func() {
				resetManager()
				resetConfig()
				expr := Expression{
					id:         1,
					status:     "done",
					result:     42,
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				}
				ManagerInstance.StoreExpression(1, expr)
			},
			expectedStatusCode: http.StatusOK,
			expectedResponse: &ExpressionResponse{
				ID:     1,
				Status: "done",
				Result: 42,
			},
		},
		{
			name:         "non-existing expression",
			expressionID: "999",
			setupExpression: func() {
				resetManager()
				resetConfig()
			},
			expectedStatusCode: http.StatusNotFound,
			expectedResponse:   nil,
		},
		{
			name:         "invalid ID format",
			expressionID: "abc",
			setupExpression: func() {
				resetManager()
				resetConfig()
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedResponse:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupExpression()

			req, err := http.NewRequest(http.MethodGet, "/expressions/"+tt.expressionID, nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			vars := map[string]string{
				"id": tt.expressionID,
			}
			req = mux.SetURLVars(req, vars)

			HandleGetExpressionByID(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedStatusCode == http.StatusOK {
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

				var response ExpressionResponse
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResponse.ID, response.ID)
				assert.Equal(t, tt.expectedResponse.Status, response.Status)
				assert.Equal(t, tt.expectedResponse.Result, response.Result)
			}
		})
	}
}

func TestHandleGetTask(t *testing.T) {
	tests := []struct {
		name               string
		setupQueue         func()
		expectedStatusCode int
		expectedTask       *TaskResponse
	}{
		{
			name: "empty queue",
			setupQueue: func() {
				resetManager()
				resetConfig()
			},
			expectedStatusCode: http.StatusNotFound,
			expectedTask:       nil,
		},
		{
			name: "task available",
			setupQueue: func() {
				resetManager()
				resetConfig()
				config.AppConfig.TimeAddition = 1 * time.Second
				config.AppConfig.TimeSubtraction = 1 * time.Second
				config.AppConfig.TimeMultiplication = 2 * time.Second
				config.AppConfig.TimeDivision = 3 * time.Second

				expr := Expression{
					id:         1,
					operator:   "+",
					status:     "waiting",
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				}
				expr.leftValue <- 5
				expr.rightValue <- 3

				ManagerInstance.StoreExpression(1, expr)
				ManagerInstance.AddTask(1)
			},
			expectedStatusCode: http.StatusOK,
			expectedTask: &TaskResponse{
				Task: TaskResponseArgs{
					ID:            1,
					LeftValue:     5,
					RightValue:    3,
					Operator:      "+",
					OperationTime: 1 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupQueue()

			req, err := http.NewRequest(http.MethodGet, "/task", nil)
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			HandleGetTask(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedStatusCode == http.StatusOK {
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

				var task TaskResponse
				err = json.Unmarshal(rr.Body.Bytes(), &task)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedTask.Task.ID, task.Task.ID)
				assert.Equal(t, tt.expectedTask.Task.LeftValue, task.Task.LeftValue)
				assert.Equal(t, tt.expectedTask.Task.RightValue, task.Task.RightValue)
				assert.Equal(t, tt.expectedTask.Task.Operator, task.Task.Operator)
				assert.Equal(t, tt.expectedTask.Task.OperationTime, task.Task.OperationTime)
			}
		})
	}
}

func TestHandlePostTaskResult(t *testing.T) {
	tests := []struct {
		name               string
		setupExpression    func()
		requestBody        string
		expectedStatusCode int
		mockProcessResult  func(result TaskResult) error
	}{
		{
			name: "valid result",
			setupExpression: func() {
				resetManager()
				resetConfig()
				expr := Expression{
					id:         1,
					isRoot:     true,
					status:     "working",
					leftValue:  make(chan float64, 1),
					rightValue: make(chan float64, 1),
				}
				ManagerInstance.StoreExpression(1, expr)
			},
			requestBody:        `{"id":1,"result":42,"error":"nil"}`,
			expectedStatusCode: http.StatusOK,
			mockProcessResult:  nil,
		},
		{
			name: "invalid JSON body",
			setupExpression: func() {
				resetManager()
				resetConfig()
			},
			requestBody:        `{"id":1,"result":42,"error"nil"}`,
			expectedStatusCode: http.StatusUnprocessableEntity,
			mockProcessResult:  nil,
		},
		{
			name: "expression not found",
			setupExpression: func() {
				resetManager()
				resetConfig()
			},
			requestBody:        `{"id":999,"result":42,"error":"nil"}`,
			expectedStatusCode: http.StatusNotFound,
			mockProcessResult: func(result TaskResult) error {
				return ErrExpressionNotFound
			},
		},
		{
			name: "internal server error",
			setupExpression: func() {
				resetManager()
				resetConfig()
			},
			requestBody:        `{"id":1,"result":42,"error":"nil"}`,
			expectedStatusCode: http.StatusNotFound,
			mockProcessResult: func(result TaskResult) error {
				return errors.New("some internal error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupExpression()

			req, err := http.NewRequest(http.MethodPost, "/task/result", bytes.NewBufferString(tt.requestBody))
			require.NoError(t, err)

			rr := httptest.NewRecorder()

			HandlePostTaskResult(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)
		})
	}
}
