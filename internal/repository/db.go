package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/wb-go/wbf/dbpg"
	"github.com/wb-go/wbf/zlog"

	"ImageProcessor/internal/model"
)

type Storager interface {
	CreateImage(ctx context.Context, img model.ImageInCreate) (int, error)
	GetImage(ctx context.Context, id int) (model.ImageInRepo, error)
	UpdateImage(ctx context.Context, img model.ImageInRepo) error
	DeleteImage(ctx context.Context, id int) error

	GetImages(ctx context.Context, lastCreatedAt time.Time, lastID int, mode string) ([]model.ImageInRepo, error)
	GetCountImages(ctx context.Context) (int, error)

	Close() error
}

type Storage struct {
	DB *dbpg.DB
}

func (s *Storage) Close() error {

	return s.DB.Master.Close()
}

func NewStorage(dsn string) (*Storage, error) {
	opts := dbpg.Options{
		MaxOpenConns:    10,
		MaxIdleConns:    10,
		ConnMaxLifetime: 1 * time.Minute,
	}
	db, err := dbpg.New(dsn, nil, &opts)
	if err != nil {
		return nil, err
	}
	return &Storage{DB: db}, nil
}

func (s *Storage) DeleteImage(ctx context.Context, id int) error {
	query := `DELETE 
				FROM image_path
				WHERE id=$1`
	res, err := s.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (s *Storage) UpdateImage(ctx context.Context, img model.ImageInRepo) error {
	query := `UPDATE image_path
				SET processed_path=$1,
					processed=$2
				WHERE id=$3`
	_, err := s.DB.ExecContext(ctx, query, img.ProcessedPath, img.Processed, img.ID)
	if err != nil {
		return err
	}
	return nil
}

func (s *Storage) CreateImage(ctx context.Context, img model.ImageInCreate) (int, error) {
	query := `INSERT INTO image_path (uploads_path, processed_path, processed, created_at)
				VALUES ($1, $2, $3, $4)
				RETURNING id`
	var id int
	res := s.DB.QueryRowContext(ctx, query, img.UploadsPath, img.ProcessedPath, img.Processed, time.Now())
	if res.Err() != nil {
		return 0, res.Err()
	}
	err := res.Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Storage) GetImage(ctx context.Context, id int) (model.ImageInRepo, error) {
	query := `SELECT id, uploads_path, processed_path, processed, created_at
				FROM image_path
				WHERE id=$1`
	res, err := s.DB.QueryContext(ctx, query, id)
	if err != nil {
		return model.ImageInRepo{}, err
	}

	defer func() {
		if err := res.Close(); err != nil {
			zlog.Logger.Error().Msg(err.Error())
		}
	}()

	var img model.ImageInRepo
	if res.Next() {
		err := res.Scan(&img.ID, &img.UploadsPath, &img.ProcessedPath, &img.Processed, &img.CreatedAt)
		if err != nil {
			return model.ImageInRepo{}, err
		}
	}
	return img, nil
}

func (s *Storage) GetImages(ctx context.Context, lastCreatedAt time.Time, lastID int, mode string) ([]model.ImageInRepo, error) {
	var query string
	var args []interface{}

	switch mode {
	case "next":
		query = `SELECT id, uploads_path, processed_path, processed, created_at
                FROM image_path
                WHERE created_at > $1 AND id > $2
                ORDER BY created_at ASC, id ASC
                LIMIT 4`
		args = []interface{}{lastCreatedAt, lastID}

	case "prev":
		query = `SELECT id, uploads_path, processed_path, processed, created_at
                FROM image_path
                WHERE (created_at < $1) OR (created_at = $1 AND id < $2)
                ORDER BY created_at DESC, id DESC
                LIMIT 4`
		args = []interface{}{lastCreatedAt, lastID}
	}

	res, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer res.Close()

	images := make([]model.ImageInRepo, 0)
	for res.Next() {
		var temp model.ImageInRepo
		err := res.Scan(&temp.ID, &temp.UploadsPath, &temp.ProcessedPath, &temp.Processed, &temp.CreatedAt)
		if err != nil {
			return nil, err
		}
		images = append(images, temp)
	}
	return images, nil
}

func (s *Storage) GetCountImages(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM image_path`
	res := s.DB.QueryRowContext(ctx, query)
	if err := res.Err(); err != nil {
		return 0, err
	}

	var count int
	err := res.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
