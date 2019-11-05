package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/kvzoo"
	pb "github.com/zdnscloud/kvzoo/proto"
)

type Proxy struct {
	master *Client
	slaves []*Client
}

const (
	ConnectTimeout = 10 * time.Second
	InvalidTxID    = int64(-1)
)

func New(masterAddr string, slaveAddrs []string) (kvzoo.DB, error) {
	master, err := NewClient(masterAddr, ConnectTimeout)
	if err != nil {
		return nil, err
	}

	slaves := make([]*Client, 0, len(slaveAddrs))
	for _, addr := range slaveAddrs {
		slave, err := NewClient(addr, ConnectTimeout)
		if err != nil {
			return nil, err
		}
		slaves = append(slaves, slave)
	}

	return &Proxy{
		master: master,
		slaves: slaves,
	}, nil
}

func (p *Proxy) Checksum() (string, error) {
	req := &pb.ChecksumRequest{}
	reply, err := p.master.Checksum(context.TODO(), req)
	if err != nil {
		return "", err
	}

	cs := reply.Checksum
	for _, slave := range p.slaves {
		if reply, err := slave.Checksum(context.TODO(), req); err != nil {
			return "", fmt.Errorf("%s get checksum failed:%s", slave.Target(), err.Error())
		} else if reply.Checksum != cs {
			return "", fmt.Errorf("checksum of %s isn't same with master %s", slave.Target(), p.master.Target())
		}
	}
	return cs, nil
}

func (p *Proxy) Close() error {
	var err error
	if err_ := p.master.Close(); err_ != nil {
		err = err_
	}

	for _, slave := range p.slaves {
		if err_ := slave.Close(); err == nil && err_ != nil {
			err = err_
		}
	}

	return err
}

func (p *Proxy) Destroy() error {
	req := &pb.DestroyRequest{}
	if _, err := p.master.Destroy(context.TODO(), req); err != nil {
		return err
	}

	for _, slave := range p.slaves {
		if _, err := slave.Destroy(context.TODO(), req); err != nil {
			log.Warnf("%s Destroy failed:%s", slave.Target(), err.Error())
		}
	}
	return p.Close()
}

type ProxyTable struct {
	proxy     *Proxy
	tableName string
}

func (p *Proxy) CreateOrGetTable(tableName kvzoo.TableName) (kvzoo.Table, error) {
	req := &pb.CreateOrGetTableRequest{
		Name: string(tableName),
	}

	if _, err := p.master.CreateOrGetTable(context.TODO(), req); err != nil {
		return nil, err
	}

	for _, slave := range p.slaves {
		if _, err := slave.CreateOrGetTable(context.TODO(), req); err != nil {
			log.Warnf("%s CreateOrGetTable failed:%s", slave.Target(), err.Error())
		}
	}

	return &ProxyTable{
		proxy:     p,
		tableName: string(tableName),
	}, nil
}

func (p *Proxy) DeleteTable(tableName kvzoo.TableName) error {
	req := &pb.DeleteTableRequest{
		Name: string(tableName),
	}

	if _, err := p.master.DeleteTable(context.TODO(), req); err != nil {
		return err
	}

	for _, slave := range p.slaves {
		if _, err := slave.DeleteTable(context.TODO(), req); err != nil {
			log.Warnf("%s DeleteTable failed:%s", slave.Target(), err.Error())
		}
	}
	return nil
}

type ProxyTransaction struct {
	proxy *Proxy
	ids   []int64
}

func (tb *ProxyTable) Begin() (kvzoo.Transaction, error) {
	req := &pb.BeginTransactionRequest{
		TableName: tb.tableName,
	}

	p := tb.proxy
	var tx *ProxyTransaction
	if reply, err := p.master.BeginTransaction(context.TODO(), req); err != nil {
		return nil, err
	} else {
		ids := make([]int64, 0, 1+len(p.slaves))
		ids = append(ids, reply.TxId)
		tx = &ProxyTransaction{
			proxy: tb.proxy,
			ids:   ids,
		}
	}

	for _, slave := range p.slaves {
		if reply, err := slave.BeginTransaction(context.TODO(), req); err != nil {
			log.Warnf("%s BeginTransaction failed:%s", slave.Target(), err.Error())
			tx.ids = append(tx.ids, InvalidTxID)
		} else {
			tx.ids = append(tx.ids, reply.TxId)
		}
	}

	return tx, nil
}

func (tx *ProxyTransaction) Rollback() error {
	req := &pb.RollbackTransactionRequest{
		TxId: tx.ids[0],
	}

	p := tx.proxy
	if _, err := p.master.RollbackTransaction(context.TODO(), req); err != nil {
		return err
	}

	for i, slave := range p.slaves {
		id := tx.ids[i+1]
		if id == InvalidTxID {
			continue
		}

		req := &pb.RollbackTransactionRequest{
			TxId: id,
		}
		if _, err := slave.RollbackTransaction(context.TODO(), req); err != nil {
			log.Warnf("%s Rollback failed:%s", slave.Target(), err.Error())
		}
	}
	return nil
}

func (tx *ProxyTransaction) Commit() error {
	req := &pb.CommitTransactionRequest{
		TxId: tx.ids[0],
	}

	p := tx.proxy
	if _, err := p.master.CommitTransaction(context.TODO(), req); err != nil {
		return err
	}

	for i, slave := range p.slaves {
		id := tx.ids[i+1]
		if id == InvalidTxID {
			continue
		}

		req := &pb.CommitTransactionRequest{
			TxId: id,
		}
		if _, err := slave.CommitTransaction(context.TODO(), req); err != nil {
			log.Warnf("%s commit failed:%s", slave.Target(), err.Error())
		}
	}
	return nil
}

func (tx *ProxyTransaction) Add(key string, value []byte) error {
	req := &pb.AddRequest{
		TxId:  tx.ids[0],
		Key:   key,
		Value: value,
	}
	p := tx.proxy
	if _, err := p.master.Add(context.TODO(), req); err != nil {
		return err
	}

	for i, slave := range p.slaves {
		id := tx.ids[i+1]
		if id == InvalidTxID {
			continue
		}

		req := &pb.AddRequest{
			TxId:  id,
			Key:   key,
			Value: value,
		}
		if _, err := slave.Add(context.TODO(), req); err != nil {
			log.Warnf("%s Add %s failed:%s", slave.Target(), key, err.Error())
		}
	}
	return nil
}

func (tx *ProxyTransaction) Delete(key string) error {
	req := &pb.DeleteRequest{
		TxId: tx.ids[0],
		Key:  key,
	}
	p := tx.proxy
	if _, err := p.master.Delete(context.TODO(), req); err != nil {
		return err
	}

	for i, slave := range p.slaves {
		id := tx.ids[i+1]
		if id == InvalidTxID {
			continue
		}

		req := &pb.DeleteRequest{
			TxId: id,
			Key:  key,
		}
		if _, err := slave.Delete(context.TODO(), req); err != nil {
			log.Warnf("%s delete %s failed:%s", slave.Target(), key, err.Error())
		}
	}
	return nil
}

func (tx *ProxyTransaction) Update(key string, value []byte) error {
	req := &pb.UpdateRequest{
		TxId:  tx.ids[0],
		Key:   key,
		Value: value,
	}
	p := tx.proxy
	if _, err := p.master.Update(context.TODO(), req); err != nil {
		return err
	}

	for i, slave := range p.slaves {
		id := tx.ids[i+1]
		if id == InvalidTxID {
			continue
		}

		req := &pb.UpdateRequest{
			TxId:  id,
			Key:   key,
			Value: value,
		}
		if _, err := slave.Update(context.TODO(), req); err != nil {
			log.Warnf("%s Update %s failed:%s", slave.Target(), key, err.Error())
		}
	}
	return nil
}

func (tx *ProxyTransaction) Get(key string) ([]byte, error) {
	req := &pb.GetRequest{
		TxId: tx.ids[0],
		Key:  key,
	}
	if reply, err := tx.proxy.master.Get(context.TODO(), req); err != nil {
		if strings.Contains(err.Error(), kvzoo.ErrNotFound.Error()) {
			return nil, kvzoo.ErrNotFound
		} else {
			return nil, err
		}
	} else {
		return reply.Value, nil
	}
}

func (tx *ProxyTransaction) List() (map[string][]byte, error) {
	req := &pb.ListRequest{
		TxId: tx.ids[0],
	}
	if reply, err := tx.proxy.master.List(context.TODO(), req); err != nil {
		return nil, err
	} else {
		return reply.Values, nil
	}
}
