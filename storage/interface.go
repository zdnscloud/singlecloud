package storage

type Storage interface {
	Close() error
	CreateOrGetTable(string) (Table, error)
	DeleteTable(string) error
}

type Table interface {
	Begin() (Transaction, error)
}

type Transaction interface {
	Commit() error
	Rollback() error

	Add(string, []byte) error
	Delete(string) error
	Update(string, []byte) error
	Get(string) ([]byte, error)
	List() (map[string][]byte, error)
}
