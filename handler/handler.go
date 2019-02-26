package handler

import (
	"github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/logger"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Create(obj types.Object) (interface{}, error) {
	logger.GetLogger().Debug("create %s %s", obj.GetType(), obj.GetID())
	return nil, nil
}

func (h *Handler) Delete(obj types.Object) error {
	logger.GetLogger().Debug("delete %s %s", obj.GetType(), obj.GetID())
	return nil
}

func (h *Handler) BatchDelete(typ types.ObjectType) error {
	logger.GetLogger().Debug("delete all %s", typ.GetType())
	return nil
}

func (h *Handler) Update(objTyp types.ObjectType, objId types.ObjectID, obj types.Object) (interface{}, error) {
	logger.GetLogger().Debug("update %s %s", objTyp.GetType(), objId.GetID())
	return nil, nil
}

func (h *Handler) List(typ types.ObjectType) interface{} {
	logger.GetLogger().Debug("get all %s", typ.GetType())
	return nil
}

func (h *Handler) Get(obj types.Object) interface{} {
	logger.GetLogger().Debug("get %s %s", obj.GetType(), obj.GetID())
	return nil
}

func (h *Handler) Action(obj types.Object, action string, params map[string]interface{}) (interface{}, error) {
	logger.GetLogger().Debug("run action %s with params %v for %s %s", action, params, obj.GetType(), obj.GetID())
	return nil, nil
}
