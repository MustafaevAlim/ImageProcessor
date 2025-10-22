package imagetest

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ImageProcessor/internal/api/handlers"
	"ImageProcessor/internal/model"
	"ImageProcessor/internal/repository/mocks"
)

type InputData struct {
	id string
}

func TestGetImage(t *testing.T) {
	tests := []struct {
		name           string
		setupMock      func(*mocks.MockStorager, *mocks.MockImageStore, int)
		in             InputData
		expectedStatus int
		expectedData   map[string]string
	}{
		{
			name: "get processed image",
			setupMock: func(db *mocks.MockStorager, is *mocks.MockImageStore, id int) {
				img := model.ImageInRepo{
					ID:            id,
					UploadsPath:   "test/test.png",
					ProcessedPath: "test/processed/test.png",
					CreatedAt:     time.Date(2025, 10, 21, 12, 0, 0, 0, time.UTC),
					Processed:     true,
				}
				db.On("GetImage", mock.Anything, id).Return(img, nil).Once()
				is.On("GetURL", mock.Anything, img).Return(img.ProcessedPath, nil).Once()
			},
			in:             InputData{id: "10"},
			expectedStatus: http.StatusOK,
			expectedData: map[string]string{
				"result": "test/processed/test.png",
			},
		},
		{
			name: "get not processed image",
			setupMock: func(db *mocks.MockStorager, is *mocks.MockImageStore, id int) {
				img := model.ImageInRepo{
					ID:            id,
					UploadsPath:   "test/test.png",
					ProcessedPath: "test/processed/test.png",
					CreatedAt:     time.Date(2025, 10, 21, 12, 0, 0, 0, time.UTC),
					Processed:     false,
				}
				db.On("GetImage", mock.Anything, id).Return(img, nil).Once()
			},
			in:             InputData{id: "10"},
			expectedStatus: http.StatusAccepted,
			expectedData: map[string]string{
				"result": "image processing",
			},
		},
		{
			name: "not found image",
			setupMock: func(db *mocks.MockStorager, is *mocks.MockImageStore, id int) {
				db.On("GetImage", mock.Anything, id).Return(model.ImageInRepo{}, sql.ErrNoRows).Once()
			},
			in:             InputData{id: "20"},
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

			id, err := strconv.Atoi(tt.in.id)
			require.NoError(t, err)

			tt.setupMock(mockDB, mockImageService, id)

			h := handlers.NewHandler(mockDB, nil, mockImageService)

			rr := httptest.NewRecorder()
			g, _ := gin.CreateTestContext(rr)
			g.Request = httptest.NewRequest("GET", fmt.Sprintf("/image/%s", tt.in.id), nil)
			g.Params = gin.Params{
				gin.Param{Key: "id", Value: tt.in.id},
			}

			h.GetImage(g)

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
