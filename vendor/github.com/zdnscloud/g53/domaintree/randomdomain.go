package domaintree

import (
	"bytes"
	"math/rand"

	"github.com/zdnscloud/cement/randomdata"
	"github.com/zdnscloud/g53"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyz0123456789-"

func RandomDomain() string {
	domainLen := rand.Intn(g53.MAX_WIRE - 1)
	if domainLen == 0 {
		domainLen = 1
	}
	labelCnt := rand.Intn(g53.MAX_LABELS)
	if labelCnt == 0 {
		labelCnt = 1
	}
	generatedLen := 0
	var buf bytes.Buffer
	for i := 0; i < labelCnt; i++ {
		maxLabelLen := g53.MAX_LABEL_LEN
		if len := domainLen - generatedLen; len < maxLabelLen {
			if len > 0 {
				maxLabelLen = len
			} else {
				maxLabelLen = 1
			}
		}

		labelLen := rand.Intn(maxLabelLen)
		if labelLen == 0 {
			labelLen = 1
		}
		generatedLen = generatedLen + labelLen + 1
		if generatedLen > domainLen {
			break
		}
		randomDomain := randomdata.RandStringWithLetter(labelLen, letterBytes)
		buf.WriteString(randomDomain)
		buf.WriteString(".")
	}

	if buf.Len() == 0 {
		buf.WriteString(randomdata.RandStringWithLetter(1, letterBytes))
		buf.WriteString(".")
	}

	return buf.String()
}
