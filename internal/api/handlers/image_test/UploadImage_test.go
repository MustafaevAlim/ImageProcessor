package imagetest

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ImageProcessor/internal/api/handlers"
	"ImageProcessor/internal/repository/mocks"
)

const (
	testImagePath             = "testdata/testimage.jpg"
	testWatermarkPath         = "testdata/watermark.png"
	testInvalidDataFormatPath = "testdata/invalid.txt"
)

type Parameters struct {
	typeProcessing string
	inputFilePath  string
	height         string
	width          string
	watermarkPath  string
}

func createMultipartRequest(t *testing.T, filePath string, param Parameters) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	err = writer.WriteField("type_processing", param.typeProcessing)
	require.NoError(t, err)
	switch param.typeProcessing {
	case "resize":
		err = writer.WriteField("height", param.height)
		require.NoError(t, err)
		err = writer.WriteField("width", param.width)
		require.NoError(t, err)
	case "watermark":
		watermarkFile, err := os.Open(param.watermarkPath)
		require.NoError(t, err)
		part, err := writer.CreateFormFile("watermark", filepath.Base(watermarkFile.Name()))
		require.NoError(t, err)
		_, err = io.Copy(part, watermarkFile)
		require.NoError(t, err)
		watermarkFile.Close()

	}

	part, err := writer.CreateFormFile("img", filepath.Base(file.Name()))
	require.NoError(t, err)

	_, err = io.Copy(part, file)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	req := httptest.NewRequest("POST", "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}

func TestUploadImage(t *testing.T) {
	tests := []struct {
		name           string
		param          Parameters
		setupMock      func(*mocks.MockStorager, *mocks.MockImageTaskProducer, *mocks.MockImageStore)
		expectedStatus int
	}{
		{
			name: "thumbnail processing",
			param: Parameters{
				typeProcessing: "thumbnail",
				inputFilePath:  testImagePath,
			},
			setupMock: func(db *mocks.MockStorager, prod *mocks.MockImageTaskProducer, is *mocks.MockImageStore) {
				db.On("CreateImage", mock.Anything, mock.Anything).Return(1, nil).Once()
				prod.On("Publish", mock.Anything, mock.Anything).Return(nil).Once()
				is.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "resize processing",
			param: Parameters{
				typeProcessing: "resize",
				inputFilePath:  testImagePath,
				height:         "200",
				width:          "200",
			},
			setupMock: func(db *mocks.MockStorager, prod *mocks.MockImageTaskProducer, is *mocks.MockImageStore) {
				db.On("CreateImage", mock.Anything, mock.Anything).Return(1, nil).Once()
				prod.On("Publish", mock.Anything, mock.Anything).Return(nil).Once()
				is.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()

			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "watermark processing",
			param: Parameters{
				typeProcessing: "watermark",
				inputFilePath:  testImagePath,
				watermarkPath:  testWatermarkPath,
			},
			setupMock: func(db *mocks.MockStorager, prod *mocks.MockImageTaskProducer, is *mocks.MockImageStore) {
				db.On("CreateImage", mock.Anything, mock.Anything).Return(1, nil).Once()
				prod.On("Publish", mock.Anything, mock.Anything).Return(nil).Once()
				is.On("Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Times(2)

			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "invalid file format",
			param: Parameters{
				typeProcessing: "thumbnail",
				inputFilePath:  testInvalidDataFormatPath,
			},
			setupMock: func(db *mocks.MockStorager, prod *mocks.MockImageTaskProducer, is *mocks.MockImageStore) {
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDB := mocks.NewMockStorager(t)
			mockProducer := mocks.NewMockImageTaskProducer(t)
			mockImageStorage := mocks.NewMockImageStore(t)
			tt.setupMock(mockDB, mockProducer, mockImageStorage)
			h := handlers.NewHandler(mockDB, mockProducer, mockImageStorage)
			rr := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rr)
			c.Request = createMultipartRequest(t, tt.param.inputFilePath, tt.param)

			h.UploadImage(c)
			require.Equal(t, tt.expectedStatus, rr.Code)

			mockDB.AssertExpectations(t)
			mockProducer.AssertExpectations(t)

		})
	}
}
