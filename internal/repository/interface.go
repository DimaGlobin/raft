package repository

type Repository interface {
	Create(key, value string) error
	Get(key string) (string, error)
	Update(key, value string) error
	Delete(key string) error
}
