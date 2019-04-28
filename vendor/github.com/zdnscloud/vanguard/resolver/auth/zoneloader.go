package auth

import (
	"fmt"
	"strings"

	"github.com/zdnscloud/g53"
	util "github.com/zdnscloud/g53/util"
	"github.com/zdnscloud/vanguard/logger"
	z "github.com/zdnscloud/vanguard/resolver/auth/zone"
	"github.com/zdnscloud/vanguard/resolver/auth/zone/memoryzone"
)

func loadZone(origin *g53.Name, content string) z.Zone {
	zone := memoryzone.NewDynamicZone(origin)
	loadChan := make(chan *g53.RRset)
	abortChan := make(chan struct{})
	go parseZoneContent(content, loadChan)

	if err := zone.Load(loadChan, abortChan); err != nil {
		logger.GetLogger().Error("load zone %s with failed: %s", origin.String(false), err.Error())
	}

	return zone
}

func loadZoneFromMaster(origin *g53.Name, view string, masters []string) z.Zone {
	zone := memoryzone.NewDynamicZone(origin)
	zone.SetMasters(masters)
	abortChan := make(chan struct{})
	for _, master := range masters {
		loadChan := make(chan *g53.RRset)
		go func() {
			if err := doAXFR(origin, master, loadChan); err != nil {
				abortChan <- struct{}{}
			}
			close(loadChan)
		}()
		if err := zone.Load(loadChan, abortChan); err == nil {
			logger.GetLogger().Info("load zone %s with view %s from master %s succeed",
				origin.String(false), view, master)
			break
		}
	}
	return zone
}

func genAXFRQueryData(origin *g53.Name) []byte {
	render := g53.NewMsgRender()
	query := g53.MakeQuery(origin, g53.RR_AXFR, 1024, false)
	query.RecalculateSectionRRCount()
	query.Rend(render)
	return render.Data()
}

func doAXFR(zone *g53.Name, master string, loadChan chan<- *g53.RRset) error {
	conn, err := util.NewTCPConn(master)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := util.TCPWrite(genAXFRQueryData(zone), conn); err != nil {
		return err
	}

	firstPackageValid := false
	lastPackageValid := false
	for {
		buf, err := util.TCPRead(conn)
		if err != nil {
			return err
		}

		axfr, err := g53.MessageFromWire(util.NewInputBuffer(buf))
		if err != nil {
			return err
		}

		rrsets := axfr.GetSection(g53.AnswerSection)
		rrLen := len(rrsets)
		if rrLen == 0 {
			return fmt.Errorf("axfr message must not has empty answer section")
		}

		if firstPackageValid == false {
			if rrsets[0].Type != g53.RR_SOA {
				return fmt.Errorf("axfr first message has no soa")
			}
			firstPackageValid = true
		}

		if rrsets[rrLen-1].Type == g53.RR_SOA {
			lastPackageValid = true
			rrsets = rrsets[:rrLen-1]
		}

		for _, rrset := range rrsets {
			loadChan <- rrset
		}

		if lastPackageValid {
			break
		}
	}

	if firstPackageValid == false || lastPackageValid == false {
		return fmt.Errorf("axfr must begin with soa and end with soa")
	}

	return nil
}

func parseZoneContent(content string, loadChan chan<- *g53.RRset) {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r\n ")
		if line == "" {
			continue
		}

		if rrset, err := g53.RRsetFromString(line); err != nil {
			logger.GetLogger().Error("rr \"%s\" parse failed:%s", line, err.Error())
		} else if z.IsRRsetTypeSupport(rrset.Type) == false {
			logger.GetLogger().Debug("rr \"%s\" isn't supported", line)
		} else {
			loadChan <- rrset
		}
	}
	close(loadChan)
}
