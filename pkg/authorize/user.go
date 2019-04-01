package authorize

import (
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type UserManager struct {
	users map[string]types.User
}
