package viewselector

import (
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/core"
)

type ViewSelector interface {
	ReloadConfig(*config.VanguardConf)
	ViewForQuery(*core.Client) (string, bool)
	GetViews() []string
}
