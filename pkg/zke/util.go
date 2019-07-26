package zke

import (
	"fmt"

	"github.com/zdnscloud/cement/log"
)

func priintLog(logCh chan string) {
	go func() {
		for {
			l, ok := <-logCh
			if !ok {
				log.Infof("log channel has beed closed, will retrun")
				return
			}
			fmt.Printf(l)
		}
	}()
}
