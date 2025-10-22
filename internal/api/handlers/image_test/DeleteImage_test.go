package imagetest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ImageProcessor/internal/api/handlers"
	"ImageProcessor/internal/repository/mocks"
)

func TestDeleteImage(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockStorager, int)
		id             string
		expectedStatus int
		expectedData   map[string]string
	}{
		{
			name: "delete image success",
			setupMock: func(db *mocks.MockStorager, id int) {
				db.On("DeleteImage", mock.Anything, id).Return(nil).Once()
			},
			id:             "10",
			expectedStatus: http.StatusOK,
			expectedData: map[string]string{
				"result": "image delete",
			},
		},
		{
			name: "delete image not found",
			setupMock: func(db *mocks.MockStorager, id int) {
				db.On("DeleteImage", mock.Anything, id).Return(sql.ErrNoRows).Once()
			},
			id:             "10",
			expectedStatus: http.StatusNotFound,
			expectedData: map[string]string{
				"result": "not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := mocks.NewMockStorager(t)
			mockImageService := mocks.NewMockImageStore(t)

			id, err := strconv.Atoi(tt.id)
			require.NoError(t, err)

			tt.setupMock(mockDB, id)

			h := handlers.NewHandler(mockDB, nil, mockImageService)

			rr := httptest.NewRecorder()
			g, _ := gin.CreateTestContext(rr)
			g.Request = httptest.NewRequest("GET", fmt.Sprintf("/image/%s", tt.id), nil)
			g.Params = gin.Params{
				gin.Param{Key: "id", Value: tt.id},
			}

			h.DeleteImage(g)

			require.Equal(t, tt.expectedStatus, rr.Code)

			var response map[string]string
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			require.NoError(t, err)

			require.Contains(t, tt.expectedData["result"], response["result"])
			mockDB.AssertExpectations(t)
			mockImageService.AssertExpectations(t)

		})
	}
}
