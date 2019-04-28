package viewselector

import (
	"github.com/zdnscloud/g53"
	"github.com/zdnscloud/vanguard/config"
	"github.com/zdnscloud/vanguard/core"
)

type TSIGKey struct {
	View      string
	Name      string
	Secret    string
	Algorithm string
}

type TSIGKeyBasedView struct {
	keys map[string]*TSIGKey
}

func newTSIGKeyBasedView() *TSIGKeyBasedView {
	return &TSIGKeyBasedView{
		keys: make(map[string]*TSIGKey),
	}
}

func (v *TSIGKeyBasedView) ReloadConfig(conf *config.VanguardConf) {
	keys := make(map[string]*TSIGKey)
	for _, key := range conf.Views.ViewAcls {
		if key.KeyName == "" {
			continue
		}
		keyName := key.KeyName
		if _, ok := keys[keyName]; ok {
			panic("duplicate key")
		}
		keys[keyName] = &TSIGKey{
			View:      key.View,
			Name:      keyName,
			Secret:    key.KeySecret,
			Algorithm: key.KeyAlgorithm,
		}
	}
	v.keys = keys
}

func (m *TSIGKeyBasedView) ViewForQuery(client *core.Client) (string, bool) {
	req := client.Request
	if req.Tsig == nil {
		return "", false
	}

	keyName := req.Tsig.Header.Name.String(true)
	tsigCopy := *req.Tsig
	tsigCopy.MAC = nil
	if client.Response == nil {
		client.Response = req.MakeResponse()
	}
	client.Response.Tsig = &tsigCopy

	key, ok := m.keys[keyName]
	if ok == false {
		client.Response.Tsig.Error = uint16(g53.R_BADKEY)
		return "", true
	}

	if key.Algorithm != string(req.Tsig.Algorithm) {
		client.Response.Tsig.Error = uint16(g53.R_BADKEY)
		return "", true
	}

	if err := req.Tsig.VerifyTsig(req, key.Secret, nil); err != nil {
		client.Response.Tsig.Error = uint16(g53.R_BADSIG)
		return "", true
	}

	newTSIG, err := g53.NewTSIG(key.Name, key.Secret, key.Algorithm)
	if err != nil {
		panic("configure key invalid")
	}
	newTSIG.MAC = req.Tsig.MAC
	client.Response.SetTSIG(newTSIG)
	return key.View, true
}

func (m *TSIGKeyBasedView) KeyForView(view string) *TSIGKey {
	for _, key := range m.keys {
		if key.View == view {
			return key
		}
	}
	return nil
}

func (m *TSIGKeyBasedView) GetViews() []string {
	var views []string
	for _, key := range m.keys {
		views = append(views, key.View)
	}
	return views

}
