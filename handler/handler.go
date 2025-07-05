package handler

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/yseto/sabatrapd/charset"
	"github.com/yseto/sabatrapd/config"
	"github.com/yseto/sabatrapd/notification"
	"github.com/yseto/sabatrapd/oid"
	"github.com/yseto/sabatrapd/smi"
	"github.com/yseto/sabatrapd/template"

	g "github.com/gosnmp/gosnmp"
	"github.com/sleepinggenius2/gosmi/types"
)

const SnmpTrapOIDPrefix = ".1.3.6.1.6.3.1.1.4.1"

type Handler struct {
	Community string
	Debug     bool
	Traps     []*config.Trap

	Queue     *notification.Queue
	MibParser *smi.SMI
	Decoder   *charset.Decoder
}

func (h *Handler) OnNewTrap(packet *g.SnmpPacket, addr *net.UDPAddr) {
	// log.Printf("got trapdata from %s\n", addr.IP)

	if h.Community != "" && h.Community != packet.Community {
		if h.Debug {
			log.Printf("invalid community: expected %q, but received %q", h.Community, packet.Community)
		}
		return
	}

	var pad = make(map[string]string)
	var specificTrapFormat string
	var occurredAt = time.Now().Unix()
	var alertLevel string

	for _, v := range packet.Variables {
		if strings.HasPrefix(v.Name, SnmpTrapOIDPrefix) {
			value := v.Value.(string)

			poid, err := oid.Parse(value)
			if err != nil {
				fmt.Printf("%+v\n", err)
				continue
			}

			for i := range h.Traps {
				if oid.HasPrefix(poid, h.Traps[i].ParsedIdent) && specificTrapFormat == "" {
					specificTrapFormat = h.Traps[i].Format
					alertLevel = h.Traps[i].AlertLevel
				}
			}
		}

		var padKey, padValue string
		padKey = v.Name
		node, err := h.MibParser.FromOID(v.Name)
		if err != nil {
			fmt.Printf("%+v\n", err)
		} else {
			if node != nil {
				padKey = node.Node.RenderQualified()
			}

			if node.Node.Type != nil && node.Node.Type.BaseType == types.BaseTypeEnum {
				i, ok := v.Value.(int)
				if ok {
					padValue = node.Node.Type.Enum.Name(int64(i))
				}
			}
		}
		if padValue == "" {
			switch v.Type {
			case g.OctetString:
				b := v.Value.([]byte)
				padValue, err = h.Decoder.Decode(addr.IP.String(), b)
				if err != nil {
					fmt.Printf("%+v\n", err)
					padValue = "<cannot decode>"
				}
				// fmt.Printf("OID: %s, string: %s\n", v.Name, string(b))
			case g.ObjectIdentifier:
				valNode, err := h.MibParser.FromOID(v.Value.(string))
				if err != nil {
					fmt.Printf("%+v\n", err)
					padValue = v.Value.(string)
				} else {
					padValue = valNode.Node.Name
				}
				// fmt.Printf("OID: %s, value: %s ObjectIdentifier: %s\n", v.Name, v.Value.(string), valNode.Node.Name)
			default:
				// fmt.Printf("trap: %+v\n", v)
				padValue = fmt.Sprintf("%v", v.Value)
			}
		}

		if padKey != "" && padValue != "" {
			pad[padKey] = padValue
		}
	}

	if specificTrapFormat == "" {
		if h.Debug {
			var values []string
			for k, v := range pad {
				values = append(values, fmt.Sprintf("%q:%q", k, v))
			}
			log.Printf("skip because nothing template. format values:[%s]\n", strings.Join(values, ", "))
		}
		return
	}

	message, err := template.Execute(specificTrapFormat, pad, addr.IP.String())
	if err != nil {
		log.Println(err)
		return
	}

	h.Queue.Enqueue(notification.Item{
		OccurredAt: occurredAt,
		Addr:       addr.IP.String(),
		Message:    message,
		AlertLevel: alertLevel,
	})
}
