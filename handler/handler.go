package handler

import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/mackerelio-labs/sabatrapd/charset"
	"github.com/mackerelio-labs/sabatrapd/config"
	"github.com/mackerelio-labs/sabatrapd/notification"
	"github.com/mackerelio-labs/sabatrapd/oid"
	"github.com/mackerelio-labs/sabatrapd/smi"
	"github.com/mackerelio-labs/sabatrapd/template"

	g "github.com/gosnmp/gosnmp"
	"github.com/sleepinggenius2/gosmi/types"
)

const SnmpTrapOIDPrefix = ".1.3.6.1.6.3.1.1.4.1"

type Handler struct {
	Community string
	Traps     []*config.Trap

	Queue     *notification.Queue
	MibParser *smi.SMI
	Decoder   *charset.Decoder
}

func (h *Handler) OnNewTrap(packet *g.SnmpPacket, addr *net.UDPAddr) {
	// log.Printf("got trapdata from %s\n", addr.IP)

	if h.Community != "" && h.Community != packet.Community {
		slog.Warn("invalid community", "expected", h.Community, "received", packet.Community)
		return
	}

	var pad = make(map[string]string)
	var hasTrappedOIDs []string
	var specificTrapFormat string
	var occurredAt = time.Now().Unix()
	var alertLevel string

	for _, v := range packet.Variables {
		if strings.HasPrefix(v.Name, SnmpTrapOIDPrefix) {
			value := v.Value.(string)
			hasTrappedOIDs = append(hasTrappedOIDs, value)

			poid, err := oid.Parse(value)
			if err != nil {
				slog.Warn("failed oid.Parse", "error", err.Error())
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
			slog.Warn("failed MibParser.FromOID", "error", err.Error())
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
					slog.Warn("failed Decoder.Decode", "error", err.Error())
					padValue = "<cannot decode>"
				}
				// fmt.Printf("OID: %s, string: %s\n", v.Name, string(b))
			case g.ObjectIdentifier:
				valNode, err := h.MibParser.FromOID(v.Value.(string))
				if err != nil {
					slog.Warn("failed Decoder.Decode", "error", err.Error())
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
		var values []any
		values = append(values, slog.Any("has trapped OIDs", hasTrappedOIDs))
		for k, v := range pad {
			values = append(values, slog.String(k, v))
		}
		slog.Info("skip because nothing template", values...)
		return
	}

	message, err := template.Execute(specificTrapFormat, pad, addr.IP.String())
	if err != nil {
		slog.Warn("failed generate message", "error", err.Error())
		return
	}

	h.Queue.Enqueue(notification.Item{
		OccurredAt: occurredAt,
		Addr:       addr.IP.String(),
		Message:    message,
		AlertLevel: alertLevel,
	})
}
