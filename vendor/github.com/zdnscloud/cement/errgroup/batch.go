package errgroup

import (
	"errors"
	"reflect"
)

var (
	ErrInvalidParameter = errors.New("parameter isn't a slice")
)

func createQueueFromSlice(slice interface{}) (int, chan interface{}, error) {
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() != reflect.Slice {
		return 0, nil, ErrInvalidParameter
	}

	size := sliceVal.Len()
	if size == 0 {
		return 0, nil, nil
	}

	ch := make(chan interface{}, size)
	for i := 0; i < size; i++ {
		ch <- sliceVal.Index(i).Interface()
	}
	close(ch)
	return size, ch, nil
}

func Batch(tasks interface{}, worker func(interface{}) (interface{}, error)) (<-chan interface{}, error) {
	workerCount, taskCh, err := createQueueFromSlice(tasks)
	if err != nil {
		return nil, err
	}

	var group Group
	resultCh := make(chan interface{}, workerCount)
	for i := 0; i < workerCount; i++ {
		group.Go(func() error {
			for task := range taskCh {
				if result, err := worker(task); err != nil {
					return err
				} else {
					resultCh <- result
				}
			}
			return nil
		})
	}
	err = group.Wait()
	close(resultCh)
	return resultCh, err
}

func BatchBackgroud(tasks interface{}, worker func(interface{}) error) error {
	workerCount, taskCh, err := createQueueFromSlice(tasks)
	if err != nil {
		return err
	}

	var group Group
	for i := 0; i < workerCount; i++ {
		group.Go(func() error {
			for task := range taskCh {
				if err := worker(task); err != nil {
					return err
				}
			}
			return nil
		})
	}
	return group.Wait()
}
