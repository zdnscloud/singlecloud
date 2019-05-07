package core

type DNSQueryHandler interface {
	HandleQuery(*Context)
	Next() DNSQueryHandler
	SetNext(DNSQueryHandler)
}

type DefaultHandler struct {
	next DNSQueryHandler
}

func (h *DefaultHandler) HandleQuery(ctx *Context)     {}
func (h *DefaultHandler) Next() DNSQueryHandler        { return h.next }
func (h *DefaultHandler) SetNext(next DNSQueryHandler) { h.next = next }

func BuildQueryChain(handlers ...DNSQueryHandler) {
	var prev DNSQueryHandler
	for i, handler := range handlers {
		if i > 0 {
			prev.SetNext(handler)
		}
		prev = handler
	}
}

func PassToNext(h DNSQueryHandler, ctx *Context) {
	if next := h.Next(); next != nil {
		next.HandleQuery(ctx)
	}
}
