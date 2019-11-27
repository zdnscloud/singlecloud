package server

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/ptypes/empty"

	"github.com/zdnscloud/kvzoo"
	pb "github.com/zdnscloud/kvzoo/proto"
)

const MaxOpenTxCount = 2000

type KVService struct {
	db       kvzoo.DB
	nextTxId int64

	openedTables map[string]kvzoo.Table
	tableLock    sync.RWMutex

	openedTxs map[int64]kvzoo.Transaction
	txLock    sync.RWMutex
}

func newKVService(db kvzoo.DB) *KVService {
	return &KVService{
		db:           db,
		nextTxId:     0,
		openedTables: make(map[string]kvzoo.Table),
		openedTxs:    make(map[int64]kvzoo.Transaction),
	}
}

func (s *KVService) Close() {
	s.db.Close()
}

func (s *KVService) Checksum(ctx context.Context, in *pb.ChecksumRequest) (*pb.ChecksumReply, error) {
	cs, err := s.db.Checksum()
	if err != nil {
		return nil, err
	} else {
		return &pb.ChecksumReply{
			Checksum: cs,
		}, nil
	}
}

func (s *KVService) Destroy(ctx context.Context, in *pb.DestroyRequest) (*empty.Empty, error) {
	s.tableLock.Lock()
	defer s.tableLock.Unlock()
	s.openedTables = make(map[string]kvzoo.Table)
	s.txLock.Lock()
	defer s.txLock.Unlock()
	s.openedTxs = make(map[int64]kvzoo.Transaction)

	if err := s.db.Close(); err != nil {
		return nil, err
	}

	if err := s.db.Destroy(); err != nil {
		return nil, err
	} else {
		return &empty.Empty{}, nil
	}
}

func (s *KVService) CreateOrGetTable(ctx context.Context, in *pb.CreateOrGetTableRequest) (*empty.Empty, error) {
	s.tableLock.Lock()
	defer s.tableLock.Unlock()

	if _, ok := s.openedTables[in.Name]; ok == false {
		tn, err := kvzoo.NewTableName(in.Name)
		if err != nil {
			return nil, err
		}

		table, err := s.db.CreateOrGetTable(tn)
		if err != nil {
			return nil, err
		}
		s.openedTables[in.Name] = table
	}

	return &empty.Empty{}, nil
}

func (s *KVService) DeleteTable(ctx context.Context, in *pb.DeleteTableRequest) (*empty.Empty, error) {
	s.tableLock.Lock()
	delete(s.openedTables, in.Name)
	s.tableLock.Unlock()

	tn, err := kvzoo.NewTableName(in.Name)
	if err != nil {
		return nil, err
	}

	if err := s.db.DeleteTable(tn); err != nil {
		return nil, err
	} else {
		return &empty.Empty{}, nil
	}
}

func (s *KVService) BeginTransaction(ctx context.Context, in *pb.BeginTransactionRequest) (*pb.BeginTransactionReply, error) {
	s.tableLock.RLock()
	defer s.tableLock.RUnlock()

	table, ok := s.openedTables[in.TableName]
	if ok == false {
		return nil, fmt.Errorf("table %s doesn't exists", in.TableName)
	}

	s.txLock.RLock()
	if len(s.openedTxs) > MaxOpenTxCount {
		s.txLock.RUnlock()
		return nil, fmt.Errorf("too many transactions are opened")
	}
	s.txLock.RUnlock()

	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	id := atomic.AddInt64(&s.nextTxId, 1)
	s.txLock.Lock()
	s.openedTxs[id] = tx
	s.txLock.Unlock()
	return &pb.BeginTransactionReply{
		TxId: id,
	}, nil
}

func (s *KVService) CommitTransaction(ctx context.Context, in *pb.CommitTransactionRequest) (*empty.Empty, error) {
	s.txLock.Lock()
	defer s.txLock.Unlock()

	tx, ok := s.openedTxs[in.TxId]
	if ok == false {
		return nil, fmt.Errorf("invalid transaction id")
	}

	err := tx.Commit()
	delete(s.openedTxs, in.TxId)
	if err != nil {
		return nil, err
	} else {
		return &empty.Empty{}, nil
	}
}

func (s *KVService) RollbackTransaction(ctx context.Context, in *pb.RollbackTransactionRequest) (*empty.Empty, error) {
	s.txLock.Lock()
	defer s.txLock.Unlock()

	tx, ok := s.openedTxs[in.TxId]
	if ok == false {
		return nil, fmt.Errorf("invalid transaction id")
	}

	err := tx.Rollback()
	delete(s.openedTxs, in.TxId)
	if err != nil {
		return nil, err
	} else {
		return &empty.Empty{}, nil
	}
}

func (s *KVService) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	s.txLock.RLock()
	defer s.txLock.RUnlock()

	tx, ok := s.openedTxs[in.TxId]
	if ok == false {
		return nil, fmt.Errorf("invalid transaction id")
	}

	value, err := tx.Get(in.Key)
	if err != nil {
		return nil, err
	}

	return &pb.GetResponse{
		Value: value,
	}, nil
}

func (s *KVService) List(ctx context.Context, in *pb.ListRequest) (*pb.ListResponse, error) {
	s.txLock.RLock()
	defer s.txLock.RUnlock()

	tx, ok := s.openedTxs[in.TxId]
	if ok == false {
		return nil, fmt.Errorf("invalid transaction id")
	}

	values, err := tx.List()
	if err != nil {
		return nil, err
	}

	return &pb.ListResponse{
		Values: values,
	}, nil
}

func (s *KVService) Add(ctx context.Context, in *pb.AddRequest) (*empty.Empty, error) {
	s.txLock.RLock()
	defer s.txLock.RUnlock()

	tx, ok := s.openedTxs[in.TxId]
	if ok == false {
		return nil, fmt.Errorf("invalid transaction id")
	}

	if err := tx.Add(in.Key, in.Value); err != nil {
		return nil, err
	} else {
		return &empty.Empty{}, nil
	}
}

func (s *KVService) Delete(ctx context.Context, in *pb.DeleteRequest) (*empty.Empty, error) {
	s.txLock.RLock()
	defer s.txLock.RUnlock()

	tx, ok := s.openedTxs[in.TxId]
	if ok == false {
		return nil, fmt.Errorf("invalid transaction id")
	}

	if err := tx.Delete(in.Key); err != nil {
		return nil, err
	} else {
		return &empty.Empty{}, nil
	}
}

func (s *KVService) Update(ctx context.Context, in *pb.UpdateRequest) (*empty.Empty, error) {
	s.txLock.RLock()
	defer s.txLock.RUnlock()

	tx, ok := s.openedTxs[in.TxId]
	if ok == false {
		return nil, fmt.Errorf("invalid transaction id")
	}

	if err := tx.Update(in.Key, in.Value); err != nil {
		return nil, err
	} else {
		return &empty.Empty{}, nil
	}
}
