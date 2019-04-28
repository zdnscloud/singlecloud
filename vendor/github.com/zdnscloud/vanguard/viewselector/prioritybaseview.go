package viewselector

import (
	"math/rand"
	"sort"
	"time"

	"github.com/zdnscloud/vanguard/core"
)

type PriorityBaseView struct {
	priorityBaseViews map[string]int
	views             []string
	viewMarks         []int
}

func newPriorityBaseView() *PriorityBaseView {
	rand.Seed(time.Now().UnixNano())
	pbv := &PriorityBaseView{}
	pbv.reload()
	return pbv
}

func (pbv *PriorityBaseView) reload() []string {
	//pbv.priorityBaseViews = priorityViews
	pbv.calculateViewMarks()
	return pbv.views
}

func (pbv *PriorityBaseView) ViewForQuery(client *core.Client) (string, bool) {
	viewCount := len(pbv.views)
	if viewCount > 0 {
		marks := pbv.viewMarks
		index := rand.Intn(marks[viewCount-1]) + 1 //Intn returns [0, n), add 1 to extent to the range to [0, n]
		viewPos := sort.Search(viewCount, func(i int) bool { return marks[i] >= index })
		return pbv.views[viewPos], true
	}

	return "", false
}

func (pbv *PriorityBaseView) calculateViewMarks() {
	pbv.views = make([]string, 0, 0)
	pbv.viewMarks = make([]int, 0, 0)
	mark := 0
	for v, p := range pbv.priorityBaseViews {
		mark += p
		pbv.views = append(pbv.views, v)
		pbv.viewMarks = append(pbv.viewMarks, mark)
	}
}

func (pbv *PriorityBaseView) GetViews() []string {
	return pbv.views
}
