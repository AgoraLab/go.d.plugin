package bind

import (
	"encoding/xml"
	"fmt"
	"github.com/netdata/go.d.plugin/pkg/web"
	"net/http"
	"net/url"
	"time"
)

type xml3Stats struct {
	Server xml3Server `xml:"server"`
	Views  []xml3View `xml:"views>view"`
}

type xml3Server struct {
	BootTime        time.Time          `xml:"boot-time"`
	ReconfigureTime time.Time          `xml:"config-time"`
	CounterGroups   []xml3CounterGroup `xml:"counters"`
}

type xml3CounterGroup struct {
	Type     string `xml:"type,attr"`
	Counters []struct {
		Name  string `xml:"name,attr"`
		Value int64  `xml:",chardata"`
	} `xml:"counter"`
}

type xml3View struct {
	Name          string             `xml:"name,attr"`
	Zones         []xml3ZoneCounter  `xml:"zones>zone"`
	CounterGroups []xml3CounterGroup `xml:"counters"`
}

//type xml3Zone struct {
//	Name  string            `xml:"name,attr"`
//	Zones []xml3ZoneCounter `xml:"zones>zone"`
//}

type xml3ZoneCounter struct {
	Name       string `xml:"name,attr"`
	Rdataclass string `xml:"rdataclass,attr"`
	Type       string `xml:"type"`
	Serial     int64  `xml:"serial"`
}

func newXML3Client(client *http.Client, request web.Request) *xml3Client {
	return &xml3Client{httpClient: client, request: request}
}

type xml3Client struct {
	httpClient *http.Client
	request    web.Request
}

func (c xml3Client) serverStats() (*serverStats, error) {
	req := c.request.Copy()
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, fmt.Errorf("error on parsing URL: %v", err)
	}

	req.URL = u.String()

	httpReq, err := web.NewHTTPRequest(req)
	if err != nil {
		return nil, fmt.Errorf("error on creating HTTP request: %v", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("error on request : %v", err)
	}
	defer closeBody(resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s returned HTTP status %d", httpReq.URL, resp.StatusCode)
	}

	stats := xml3Stats{}
	if err = xml.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("error on decoding response from %s : %v", httpReq.URL, err)
	}
	return convertXML(stats), nil
}

func convertXML(xmlStats xml3Stats) *serverStats {
	stats := serverStats{
		OpCodes:         make(map[string]int64),
		NSStats:         make(map[string]int64),
		QTypes:          make(map[string]int64),
		SockStats:       make(map[string]int64),
		Views:           make(map[string]jsonView),
		BootTime:        xmlStats.Server.BootTime,
		ReconfigureTime: xmlStats.Server.ReconfigureTime,
	}

	var m map[string]int64

	for _, group := range xmlStats.Server.CounterGroups {
		switch group.Type {
		default:
			continue
		case "opcode":
			m = stats.OpCodes
		case "qtype":
			m = stats.QTypes
		case "nsstat":
			m = stats.NSStats
		case "sockstat":
			m = stats.SockStats
		}

		for _, v := range group.Counters {
			m[v.Name] = v.Value
		}
	}

	for _, view := range xmlStats.Views {
		stats.Views[view.Name] = jsonView{
			Resolver: jsonViewResolver{
				Stats:      make(map[string]int64),
				QTypes:     make(map[string]int64),
				CacheStats: make(map[string]int64),
			},
			Zones: make([]jsonViewZone, len(view.Zones)),
		}
		for i, z := range view.Zones {
			stats.Views[view.Name].Zones[i] = jsonViewZone{
				Name:   z.Name,
				Class:  z.Rdataclass,
				Serial: z.Serial,
				Type:   z.Type,
			}
		}

		for _, viewGroup := range view.CounterGroups {
			switch viewGroup.Type {
			default:
				continue
			case "resqtype":
				m = stats.Views[view.Name].Resolver.QTypes
			case "resstats":
				m = stats.Views[view.Name].Resolver.Stats
			case "cachestats":
				m = stats.Views[view.Name].Resolver.CacheStats
			}
			for _, viewCounter := range viewGroup.Counters {
				m[viewCounter.Name] = viewCounter.Value
			}
		}
	}
	return &stats
}
